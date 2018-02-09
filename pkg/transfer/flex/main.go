package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/hashicorp/go-multierror"
	"github.com/nerdalize/nerd/pkg/transfer"
	"github.com/pkg/errors"
)

//Operation is an action that can be performed with the flex volume.
type Operation string

const (
	//OperationInit is called when the flex volume needs to set itself up
	OperationInit = "init"

	//OperationMount is called when a volume needs to be mounted
	OperationMount = "mount"

	//OperationUnmount is called when the volume needs to be unmounted
	OperationUnmount = "unmount"
)

//Status describes the result of a flex volume action.
type Status string

const (
	//StatusSuccess is returned when the flex volume has been successfull
	StatusSuccess = "Success"
	//StatusFailure is returned when the flex volume has failed
	StatusFailure = "Failure"
	//StatusNotSupported is returned when a operation is supported
	StatusNotSupported = "Not supported"
)

//FileSystems can be used to specify a type of file system in a file.
type FileSystem string

const (
	//Standard, supported everywhere
	FileSystemExt4 FileSystem = "ext4"
)

//WriteSpace is the amount of space available for writing data.
//@TODO: Should be based on dataset size or customer details?
const WriteSpace = 100 * 1024 * 1024

//DirectoryPermissions are the permissions for directories created as part of flexvolume operation.
//@TODO: Spend more time checking if they make sense and are secure
const DirectoryPermissions = os.FileMode(0522)

//Relative paths used for flexvolume data
const (
	RelPathInput         = "input"
	RelPathFSInFile      = "volume"
	RelPathFSInFileMount = "mount"
	RelPathOptions       = "json"
)

//Output is returned by the flex volume implementation.
type Output struct {
	Status       Status       `json:"status"`
	Message      string       `json:"message"`
	Capabilities Capabilities `json:"capabilities"`
}

//MountOptions is specified whenever Kubernetes calls the mount, comes with
//the following keys: kubernetes.io/fsType, kubernetes.io/pod.name, kubernetes.io/pod.namespace
//kubernetes.io/pod.uid, kubernetes.io/pvOrVolumeName, kubernetes.io/readwrite, kubernetes.io/serviceAccount.name
type MountOptions struct {
	InputS3Key     string `json:"input/s3Key"`
	InputS3Bucket  string `json:"input/s3Bucket"`
	OutputS3Key    string `json:"output/s3Key"`
	OutputS3Bucket string `json:"output/s3Bucket"`
}

//Capabilities represents the supported features of a flex volume.
type Capabilities struct {
	Attach bool `json:"attach"`
}

//VolumeDriver can be implemented to facilitate the creation of pod volumes.
type VolumeDriver interface {
	Init() (Capabilities, error)
	Mount(mountPath string, opts MountOptions) error
	Unmount(mountPath string) error
}

//DatasetVolumes is a volume implementation that works with Nerdalize Datasets.
type DatasetVolumes struct{}

//datasetOpts describes any input and output for a volume.
type datasetOpts struct {
	Input  *transfer.Ref
	Output *transfer.Ref
}

//writeDatasetOpts writes dataset options to a JSON file.
func (volp *DatasetVolumes) writeDatasetOpts(path string, opts MountOptions) (*datasetOpts, error) {
	dsopts := &datasetOpts{}
	if opts.InputS3Key != "" {
		dsopts.Input = &transfer.Ref{
			Key:    opts.InputS3Key,
			Bucket: opts.InputS3Bucket,
		}

		if dsopts.Input.Bucket == "" {
			return nil, errors.New("input key configured without a bucket")
		}
	}

	if opts.OutputS3Key != "" {
		dsopts.Output = &transfer.Ref{
			Key:    opts.OutputS3Key,
			Bucket: opts.OutputS3Bucket,
		}

		if dsopts.Output.Bucket == "" {
			return nil, errors.New("output key configured without a bucket")
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create metadata file")
	}

	defer f.Close()

	enc := json.NewEncoder(f)
	err = enc.Encode(dsopts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to encode metadata")
	}

	return dsopts, nil
}

//readDatasetOpts reads dataset options from a JSON file.
func (volp *DatasetVolumes) readDatasetOpts(path string) (*datasetOpts, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, errors.Wrap(err, "failed to open metadata file")
	}

	defer f.Close()
	dsopts := &datasetOpts{}

	dec := json.NewDecoder(f)
	err = dec.Decode(dsopts)
	if err != nil {
		return nil, errors.Wrap(err, "failed to decode metadata")
	}

	return dsopts, nil
}

//deleteDatasetOpts deletes a JSON file containing dataset options.
func (volp *DatasetVolumes) deleteDatasetOpts(path string) error {
	err := os.Remove(path)
	return errors.Wrap(err, "failed to delete metadata file")
}

//createFSInFile creates a file with a file system inside of it that can be mounted.
func (volp *DatasetVolumes) createFSInFile(path string, filesystem FileSystem, size int64) error {
	//Create file with room to contain writable file system
	f, err := os.Create(path)
	if err != nil {
		err = errors.Wrap(err, "failed to create file system file")
		return err
	}

	err = f.Truncate(size)
	if err != nil {
		err = errors.Wrap(err, "failed to allocate file system size")
		return err
	}

	//Build file system within
	cmd := exec.Command("mkfs", "-t", string(filesystem), path)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	err = cmd.Run()
	if err != nil {
		err = errors.Wrap(errors.New(strings.TrimSpace(buf.String())), "failed to execute mkfs command")
		return err
	}

	return nil
}

//destroyFSInFile cleans up a file system in file.
func (volp *DatasetVolumes) destroyFSInFile(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		err = errors.Wrap(err, "failed to delete fs-in-file file")
	}

	return err
}

//provisionInput makes the specified input available at given path (input may be nil).
func (volp *DatasetVolumes) provisionInput(path string, input *transfer.Ref) error {
	//Create directory at path in case it doesn't exist yet
	err := os.MkdirAll(path, DirectoryPermissions)
	if err != nil {
		return errors.Wrap(err, "failed to create input directory")
	}

	//Abort if there is nothing to download to it
	if input == nil {
		return nil
	}

	//Download input to it
	var trans transfer.Transfer
	if trans, err = transfer.NewS3(&transfer.S3Conf{
		Bucket: input.Bucket,
	}); err != nil {
		return errors.Wrap(err, "failed to set up S3 transfer")
	}

	ref := &transfer.Ref{
		Bucket: input.Bucket,
		Key:    input.Key,
	}

	err = trans.Download(context.Background(), ref, path)
	if err != nil {
		return errors.Wrap(err, "failed to download data from S3")
	}

	return nil
}

//destroyInput cleans up a folder with input data.
func (volp *DatasetVolumes) destroyInput(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		err = errors.Wrap(err, "failed to destroy input directory")
	}

	return err
}

//mountFSInFile mounts an FS-in-file at the specified path.
func (volp *DatasetVolumes) mountFSInFile(volumePath string, mountPath string) error {
	//Create mount point
	err := os.Mkdir(mountPath, DirectoryPermissions)
	if err != nil {
		return errors.Wrap(err, "failed to create mount directory")
	}

	//Mount file system
	cmd := exec.Command("mount", volumePath, mountPath)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	err = cmd.Run()
	if err != nil {
		return errors.Wrap(errors.New(strings.TrimSpace(buf.String())), "failed to execute mount command")
	}

	return nil
}

//unmountFSInFile unmounts an FS-in-file and deletes the mount path.
func (volp *DatasetVolumes) unmountFSInFile(mountPath string) error {
	//Unmount
	cmd := exec.Command("umount", mountPath)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(errors.New(strings.TrimSpace(buf.String())), "failed to unmount fs-in-file")
	}

	//Delete mount path
	err = os.RemoveAll(mountPath)
	if err != nil {
		return errors.Wrap(err, "failed to delete fs-in-file mount point")
	}

	return nil
}

//mountOverlayFS mounts an OverlayFS with the given directories (upperDir and workDir may be auto-created).
func (volp *DatasetVolumes) mountOverlayFS(upperDir string, workDir string, lowerDir string, mountPath string) error {
	//Create directories in case they don't exist yet
	errs := []error{
		os.MkdirAll(upperDir, DirectoryPermissions),
		os.MkdirAll(workDir, DirectoryPermissions),
	}

	for _, err := range errs {
		if err != nil {
			return errors.Wrap(err, "failed to create directories")
		}
	}

	//Mount OverlayFS
	overlayArgs := fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lowerDir, upperDir, workDir)

	cmd := exec.Command("mount", "-t", "overlay", "overlay", "-o", overlayArgs, mountPath)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(errors.New(strings.TrimSpace(buf.String())), "failed to execute mount command")
	}

	return nil
}

//unmountOverlayFS unmounts an OverlayFS with the given directories (upperDir and workDir will be deleted).
func (volp *DatasetVolumes) unmountOverlayFS(upperDir string, workDir string, mountPath string) error {
	//Unmount OverlayFS
	cmd := exec.Command("umount", mountPath)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	err := cmd.Run()
	if err != nil {
		return errors.Wrap(errors.New(strings.TrimSpace(buf.String())), "failed to unmount overlayfs")
	}

	//Delete directories
	errs := []error{
		os.RemoveAll(upperDir),
		os.RemoveAll(workDir),
	}

	for _, err := range errs {
		if err != nil {
			return errors.Wrap(err, "failed to delete directories")
		}
	}

	return nil
}

//handleOutput uploads any output in the specified directory.
func (volp *DatasetVolumes) handleOutput(path string, output *transfer.Ref) error {
	// Nothing to do
	if output == nil {
		return nil
	}

	trans, err := transfer.NewS3(&transfer.S3Conf{
		Bucket: output.Bucket,
	})
	if err != nil {
		err = errors.Wrap(err, "failed to set up S3 transfer")
		return err
	}

	ref := &transfer.Ref{
		Bucket: output.Bucket,
		Key:    output.Key,
	}

	_, err = trans.Upload(context.Background(), ref, path)
	if err != nil {
		err = errors.Wrap(err, "failed to upload data to S3")
		return err
	}

	return nil
}

//getPath returns a path above the mountPath and unique to the dataset name.
func (volp *DatasetVolumes) getPath(mountPath string, name string) string {
	return filepath.Join(mountPath, "..", filepath.Base(mountPath)+"."+name)
}

//cleanDirectory deletes the contents of a directory, but not the directory itself.
func (volp *DatasetVolumes) cleanDirectory(path string) error {
	dir, err := os.Open(path)
	if err != nil {
		return err
	}
	defer dir.Close()

	names, err := dir.Readdirnames(-1)
	if err != nil {
		return err
	}

	for _, name := range names {
		err = os.RemoveAll(filepath.Join(path, name))
		if err != nil {
			return err
		}
	}

	return nil
}

//Init the flex volume.
func (volp *DatasetVolumes) Init() (Capabilities, error) {
	return Capabilities{Attach: false}, nil
}

//Mount the flex volume, path: '/var/lib/kubelet/pods/c911e5f7-0392-11e8-8237-32f9813bbd5a/volumes/foo~cifs/input', opts: &main.MountOptions{FSType:"", PodName:"imagemagick", PodNamespace:"default", PodUID:"c911e5f7-0392-11e8-8237-32f9813bbd5a", PVOrVolumeName:"input", ReadWrite:"rw", ServiceAccountName:"default"}
func (volp *DatasetVolumes) Mount(kubeMountPath string, opts MountOptions) (err error) {
	//Store dataset options
	dsopts, err := volp.writeDatasetOpts(volp.getPath(kubeMountPath, RelPathOptions), opts)

	defer func() {
		if err != nil {
			volp.deleteDatasetOpts(volp.getPath(kubeMountPath, RelPathOptions))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to write volume database")
	}

	//Set up input
	err = volp.provisionInput(volp.getPath(kubeMountPath, RelPathInput), dsopts.Input)

	defer func() {
		if err != nil {
			volp.destroyInput(volp.getPath(kubeMountPath, RelPathInput))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to provision input")
	}

	//Create volume to contain pod writes
	err = volp.createFSInFile(volp.getPath(kubeMountPath, RelPathFSInFile), FileSystemExt4, WriteSpace)

	defer func() {
		if err != nil {
			volp.destroyFSInFile(volp.getPath(kubeMountPath, RelPathFSInFile))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to create file system in a file")
	}

	//Mount the file system
	err = volp.mountFSInFile(
		volp.getPath(kubeMountPath, RelPathFSInFile),
		volp.getPath(kubeMountPath, RelPathFSInFileMount),
	)

	defer func() {
		if err != nil {
			volp.unmountFSInFile(volp.getPath(kubeMountPath, RelPathFSInFileMount))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to mount file system in a file")
	}

	//Set up overlay file system using input and writable fs-in-file
	err = volp.mountOverlayFS(
		filepath.Join(volp.getPath(kubeMountPath, RelPathFSInFileMount), "upper"),
		filepath.Join(volp.getPath(kubeMountPath, RelPathFSInFileMount), "work"),
		volp.getPath(kubeMountPath, RelPathInput),
		kubeMountPath,
	)

	defer func() {
		if err != nil {
			volp.unmountOverlayFS(
				filepath.Join(volp.getPath(kubeMountPath, RelPathFSInFileMount), "upper"),
				filepath.Join(volp.getPath(kubeMountPath, RelPathFSInFileMount), "work"),
				kubeMountPath,
			)
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to mount overlayfs")
	}

	return nil
}

//Unmount the flex volume.
func (volp *DatasetVolumes) Unmount(kubeMountPath string) (err error) {
	// Upload any output
	var dsopts *datasetOpts
	dsopts, err = volp.readDatasetOpts(volp.getPath(kubeMountPath, RelPathOptions))
	if err != nil {
		return errors.Wrap(err, "failed to read volume database")
	}

	err = volp.handleOutput(kubeMountPath, dsopts.Output)
	if err != nil {
		return errors.Wrap(err, "failed to upload output")
	}

	//Clean up (as much as possible)
	var result error

	err = errors.Wrap(
		volp.unmountOverlayFS(
			filepath.Join(volp.getPath(kubeMountPath, RelPathFSInFileMount), "upper"),
			filepath.Join(volp.getPath(kubeMountPath, RelPathFSInFileMount), "work"),
			kubeMountPath,
		),
		"failed to unmount overlayfs",
	)
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = errors.Wrap(
		volp.unmountFSInFile(volp.getPath(kubeMountPath, RelPathFSInFileMount)),
		"failed to unmount file system in a file",
	)
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = errors.Wrap(
		volp.destroyFSInFile(volp.getPath(kubeMountPath, RelPathFSInFile)),
		"failed to delete file system in a file",
	)
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = errors.Wrap(
		volp.destroyInput(volp.getPath(kubeMountPath, RelPathInput)),
		"failed to delete input data",
	)
	if err != nil {
		result = multierror.Append(result, err)
	}

	err = errors.Wrap(
		volp.deleteDatasetOpts(volp.getPath(kubeMountPath, RelPathOptions)),
		"failed to delete dataset",
	)
	if err != nil {
		result = multierror.Append(result, err)
	}

	return result
}

func main() {
	if len(os.Args) < 2 {
		log.Fatalf("usage: nerd-flex-volume [init|mount|unmount]")
	}

	//create the volume provider
	var volp VolumeDriver
	volp = &DatasetVolumes{}

	//setup default output data
	var err error
	output := Output{
		Status:  StatusNotSupported,
		Message: fmt.Sprintf("operation '%s' is unsupported", os.Args[1]),
	}

	//pass operations to the volume provider
	switch os.Args[1] {
	case OperationInit:
		output.Status = StatusSuccess
		output.Message = "Initialization successful"

		output.Capabilities, err = volp.Init()

	case OperationMount:
		output.Status = StatusSuccess
		output.Message = "Mount successful"

		if len(os.Args) < 4 {
			err = fmt.Errorf("expected at least 4 arguments for mount, got: %#v", os.Args)
		} else {
			opts := MountOptions{}
			err = json.Unmarshal([]byte(os.Args[3]), &opts)
			if err == nil {
				err = volp.Mount(os.Args[2], opts)
			}
		}

	case OperationUnmount:
		output.Status = StatusSuccess
		output.Message = "Unmount successful"

		if len(os.Args) < 3 {
			err = fmt.Errorf("expected at least 3 arguments for unmount, got: %#v", os.Args)
		} else {
			err = volp.Unmount(os.Args[2])
		}
	}

	//if any operations returned an error, mark as failure
	if err != nil {
		output.Status = StatusFailure
		output.Message = err.Error()
	}

	//encode the output
	enc := json.NewEncoder(os.Stdout)
	err = enc.Encode(output)
	if err != nil {
		log.Fatalf("failed to encode output: %v", err)
	}
}
