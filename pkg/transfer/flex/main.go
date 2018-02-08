package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/pkg/errors"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/nerdalize/nerd/pkg/transfer"
)

//Operation can be provided to the flex volume
type Operation string

const (
	//OperationInit is called when the flex volume needs to set itself up
	OperationInit = "init"

	//OperationMount is called when a volume needs to be mounted
	OperationMount = "mount"

	//OperationUnmount is called when the volume needs to be unmounted
	OperationUnmount = "unmount"
)

//Status describes a flex volume status
type Status string

const (
	//StatusSuccess is returned when the flex volume has been successfull
	StatusSuccess = "Success"
	//StatusFailure is returned when the flex volume has failed
	StatusFailure = "Failure"
	//StatusNotSupported is returned when a operation is supported
	StatusNotSupported = "Not supported"
)

//File systems to back file system in a file
type FileSystem string

const (
	//Standard, supported everywhere
	FileSystemExt4 = "ext4"
)

//Amount of space available for writing data
//@TODO: Should be based on dataset size or customer details?
const WriteSpace = 100 * 1024 * 1024

//Output is returned by the flex volume implementation
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

//Capabilities of the flex volume
type Capabilities struct {
	Attach bool `json:"attach"`
}

//VolumeDriver can be implemented to facilitate the creation of pod volumes
type VolumeDriver interface {
	Init() (Capabilities, error)
	Mount(mountPath string, opts MountOptions) error
	Unmount(mountPath string) error
}

//DatasetVolumes is a volume implementations that works with Nerdalize Datasets
type DatasetVolumes struct{}

type datasetOpts struct {
	Input  *transfer.Ref
	Output *transfer.Ref
}

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

func (volp *DatasetVolumes) deleteDatasetOpts(path string) error {
	err := os.Remove(path)
	return errors.Wrap(err, "failed to delete metadata file")
}

//Creates a file with a file system inside of it that can be mounted
func (volp *DatasetVolumes) createFSInFile(path string, filesystem string, size int64) error {
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
	cmd := exec.Command("mkfs", "-t", filesystem, path)
	buf := bytes.NewBuffer(nil)
	cmd.Stderr = buf
	err = cmd.Run()
	if err != nil {
		err = errors.Wrap(errors.New(strings.TrimSpace(buf.String())), "failed to execute mkfs command")
		return err
	}

	return nil
}

//Clean up file system in file
func (volp *DatasetVolumes) destroyFSInFile(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		err = errors.Wrap(err, "failed to delete fs-in-file file")
	}

	return err
}

//Make specified input available at given path (input may be nil)
func (volp *DatasetVolumes) provisionInput(path string, input *transfer.Ref) error {
	//Create directory at path in case it doesn't exist yet
	err := os.MkdirAll(path, os.FileMode(0522))
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

//Clean up input data
func (volp *DatasetVolumes) destroyInput(path string) error {
	err := os.RemoveAll(path)
	if err != nil {
		err = errors.Wrap(err, "failed to destroy input directory")
	}

	return err
}

//Mounts FS-in-file at specified path
func (volp *DatasetVolumes) mountFSInFile(volumePath string, mountPath string) error {
	//Create mount point
	err := os.Mkdir(mountPath, os.FileMode(0522))
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

//Mount OverlayFS with given directories (upperDir and workDir may be auto-created)
func (volp *DatasetVolumes) mountOverlayFS(upperDir string, workDir string, lowerDir string, mountPath string) error {
	//Create directories in case they don't exist yet
	errs := []error{
		os.MkdirAll(upperDir, os.FileMode(0522)),
		os.MkdirAll(workDir, os.FileMode(0522)),
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

//Unmount OverlayFS with given directories (upperDir and workDir will be deleted)
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

//Upload any output
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

func (volp *DatasetVolumes) getRelPath(mountPath string, name string) string {
	return filepath.Join(mountPath, "..", filepath.Base(mountPath)+"."+name)
}

//Deletes contents of a directory, but not the directory itself
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

//Init the flex volume
func (volp *DatasetVolumes) Init() (Capabilities, error) {
	return Capabilities{Attach: false}, nil
}

//Mount the flex voume, path: '/var/lib/kubelet/pods/c911e5f7-0392-11e8-8237-32f9813bbd5a/volumes/foo~cifs/input', opts: &main.MountOptions{FSType:"", PodName:"imagemagick", PodNamespace:"default", PodUID:"c911e5f7-0392-11e8-8237-32f9813bbd5a", PVOrVolumeName:"input", ReadWrite:"rw", ServiceAccountName:"default"}
func (volp *DatasetVolumes) Mount(kubeMountPath string, opts MountOptions) (err error) {
	//Store dataset options
	dsopts, err := volp.writeDatasetOpts(volp.getRelPath(kubeMountPath, "json"), opts)

	defer func() {
		if err != nil {
			volp.deleteDatasetOpts(volp.getRelPath(kubeMountPath, "json"))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to write volume database")
	}

	//Set up input
	err = volp.provisionInput(volp.getRelPath(kubeMountPath, "input"), dsopts.Input)

	defer func() {
		if err != nil {
			volp.destroyInput(volp.getRelPath(kubeMountPath, "input"))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to provision input")
	}

	//Create volume to contain pod writes
	err = volp.createFSInFile(volp.getRelPath(kubeMountPath, "volume"), FileSystemExt4, WriteSpace)

	defer func() {
		if err != nil {
			volp.destroyFSInFile(volp.getRelPath(kubeMountPath, "volume"))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to create file system in a file")
	}

	//Mount the file system
	err = volp.mountFSInFile(volp.getRelPath(kubeMountPath, "volume"), volp.getRelPath(kubeMountPath, "mount"))

	defer func() {
		if err != nil {
			volp.unmountFSInFile(volp.getRelPath(kubeMountPath, "mount"))
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to mount file system in a file")
	}

	//Set up overlay file system using input and writable fs-in-file
	err = volp.mountOverlayFS(
		filepath.Join(volp.getRelPath(kubeMountPath, "mount"), "upper"),
		filepath.Join(volp.getRelPath(kubeMountPath, "mount"), "work"),
		volp.getRelPath(kubeMountPath, "input"),
		kubeMountPath,
	)

	defer func() {
		if err != nil {
			volp.unmountOverlayFS(
				filepath.Join(volp.getRelPath(kubeMountPath, "mount"), "upper"),
				filepath.Join(volp.getRelPath(kubeMountPath, "mount"), "work"),
				kubeMountPath,
			)
		}
	}()

	if err != nil {
		return errors.Wrap(err, "failed to mount overlayfs")
	}

	return nil
}

//Unmount the flex volume
func (volp *DatasetVolumes) Unmount(kubeMountPath string) (err error) {
	// Upload any output
	var dsopts *datasetOpts
	dsopts, err = volp.readDatasetOpts(volp.getRelPath(kubeMountPath, "json"))
	if err != nil {
		return errors.Wrap(err, "failed to read volume database")
	}

	err = volp.handleOutput(kubeMountPath, dsopts.Output)
	if err != nil {
		return errors.Wrap(err, "failed to upload output")
	}

	//Clean up (as much as possible)
	errs := []error{
		errors.Wrap(
			volp.unmountOverlayFS(
				filepath.Join(volp.getRelPath(kubeMountPath, "mount"), "upper"),
				filepath.Join(volp.getRelPath(kubeMountPath, "mount"), "work"),
				kubeMountPath,
			),
			"failed to unmount overlayfs",
		),

		errors.Wrap(
			volp.unmountFSInFile(volp.getRelPath(kubeMountPath, "mount")),
			"failed to unmount file system in a file",
		),

		errors.Wrap(
			volp.destroyFSInFile(volp.getRelPath(kubeMountPath, "volume")),
			"failed to delete file system in a file",
		),

		errors.Wrap(
			volp.destroyInput(volp.getRelPath(kubeMountPath, "input")),
			"failed to delete input data",
		),

		errors.Wrap(
			volp.deleteDatasetOpts(volp.getRelPath(kubeMountPath, "json")),
			"failed to delete dataset",
		),
	}

	for _, err := range errs {
		if err != nil {
			return errors.Wrap(err, "failed to clean up")
		}
	}

	return nil
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
