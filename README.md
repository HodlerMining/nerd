# Nerdalize Scientific Compute
Your personal nerd that takes care of running scientific compute on the [Nerdalize cloud](http://nerdalize.com/cloud/).

_NOTE: This project is currently experimental and not functional._

## Command Usage

```bash
# log into the scientific compute platform
$ nerd login
Please enter your Nerdalize username and password.
Username: my-user@my-organization.com
Password: ******

# upload a piece of data that will acts as input to the program
$ nerd upload ./my-project/my-task-input
Uploading dataset with ID 'd-96fac377'
314.38 MiB / 314.38 MiB [=============================] 100.00%

# download results of running the task
$ nerd download t-615f2d56 ./my-project/my-task-output
Downloading dataset with ID 'd-615f2d56'
12.31 MiB / 12.31 MiB [=============================] 100.00%
```

Please note that each command has a `--help` option that shows how to use the command.
Each command accepts at least the following options:
```
      --config=      location of config file [$CONFIG]
  -v, --verbose      show verbose output (default: false)
      --json-format  show output in json format (default: false)
```

## Power users

### Config

The `nerd` command uses a config file located at `~/.nerd/config.json` (location can be changed with the `--config` option) which can be used to customize nerd's behaviour.
The structure of the config and the defaults are show below:
```bash
{
        "auth": {
                "api_endpoint": "http://auth.nerdalize.com", # URL of authentication server
                "public_key": "-----BEGIN PUBLIC KEY-----\nMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEAkYbLnam4wo+heLlTZEeh1ZWsfruz9nk\nkyvc4LwKZ8pez5KYY76H1ox+AfUlWOEq+bExypcFfEIrJkf/JXa7jpzkOWBDF9Sa\nOWbQHMK+vvUXieCJvCc9Vj084ABwLBgX\n-----END PUBLIC KEY-----" # Public key used to verify JWT signature
        },
        "enable_logging": false, # When set to true, all output will be logged to ~/.nerd/log
        "current_project": "", # Current project
        "nerd_token": "", # Nerdalize JWT (can be set manually or it will be set by `nerd login`)
        "nerd_api_endpoint": "https://batch.nerdalize.com" # URL of nerdalize API (NCE)
}
```

## Docker

The nerd CLI can be dockerized. To build the docker container run:

```docker build -t my-nerd .```

You can now run the container like so:

```docker run my-nerd <command>```

If you want to use your local nerd config file (which contains your credentials), you can mount it:

```docker run -v ~/.nerd:/root/.nerd my-nerd <command>```

If you just want to set your credentials, you can also set it with an environment variable:

```docker run -e NERD_JWT=put.jwt.here my-nerd <command>```

## Nerdalize SDK

Code in this repository can also be used as a Software Development Kit (SDK) to communicate with Nerdalize services. The SDK is located in the `nerd/client` package. It is devided into three different clients:

* `auth` is a client to the Nerdalize authentication backend. It can be used to fetch new JWTs.
* `batch` is a client to batch.nerdalize.com. It can be used to work with resources like `queues`, `workers`, and `datasets`.
* `data` is a client to Nerdalize storage. It can be used to upload and download datasets.
