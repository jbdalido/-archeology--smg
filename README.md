# Smuggler
======================= 

Smuggler is a simple but powerful system designed to work with docker, to help you run, test, develop and build them, make them ready to use with any docker cluster environment, or just for you ! We're also setting up a pretty simple cluster management for small usages.

Remember this is VERY **Alpha version**, and all commits are welcome to help us bring it at a new level. 


## Install

	 go get github.com/jbdalido/smg

## Command line 

Run command :

	$ - smg run --help
	NAME:
	   run - Run containers with the proper environment

	USAGE:
	   command run [command options] [arguments...]

	OPTIONS:
	   --start, -s 'smg.yml'				Specify a different file to use for your smg run (default: smg.yml)
	   --no-cache, -n					Disable the use of docker cache during run and build with provided dockerfiles
	   --verbose, -v					Verbose Mode
	   --env, -e 'default'					Environment (commands or dockerfiles) to use for the run
	   --override, -o '--override option --override option'	Environment (commands or dockerfiles) to use for the run
	   --keepalive, -k					Keep containers alive after a run (successful or not)
	   --shared-folder, -S					Use a shared-folder with the main container	

Build command : 


	bin/smg build --help
	NAME:
	   build - Build against the active git branch of the folder and the build setup of the smg file

	USAGE:
	   command build [command options] [arguments...]

	OPTIONS:
	   --start, -s 'smg.yml'		Specify a different file to use for your smg run (default: smg.yml)
	   --no-cache, -n			Disable the use of docker cache during run and build with provided dockerfiles
	   --verbose, -v			Verbose Mode
	   --push, -p				Push images after a successful build
	   --last, -l				Download last image for each build
	   --delete, -D				Delete images created after a successful build
       --tag, -t 			    Force both the action used for the build, and the image tag
	   --etcd '--etcd option --etcd option'	ETCD Storage http endpoint


## Documentation is on the way 

Alpha testers, here's some yml example of what you can do with it : 

    name: smuggler
    image: debian:jessie
    image_dockerfile: dockerfiles/deb.dockerfile

    # Open ports
    ports: 
        - 8000:80
        - 3307:33

    # Use volumes
    volumes:
        - /home/smg:/home/smg

    # Setup env variables
    env:
        - TEST=127.0.0.1

    # Use simple services
    services: 
        - mongo
        - redis
    
    # Or complex applications
    # (applications are run instead of services if exists)
    applications:
        cassandra:
            image: cassandra
            image_dockerfile: example/dockerfiles/cassandra.dockerfile
            name: cassandra
            ports:
              3305:3306

    # and run commands against it
    commands:
        default:
            - ping -c 1 cassandra
            - echo "You should write some tests"
        test:
            - echo "test party"
        make:
            - make

    # Next feature to be implemented (asap)
    # Dockerfiles will be run if exist instead of commands
    dockerfiles:
        default:
            dockerfile: dockerfiles/my.dockerfile
            entrypoint: 
                - "/bin/echo"
            cmd:
                - "test"

    # And build it once you're ready
    # Build use the Dockerfile in the current directory
    # You'll soon be able to specify it too

    # build works with regexp too
    build:
        master:
            name: local/smuggler
            # onlyif will run this command and abort if it fails. Here we’re ensuring that tests
            # pass before pushing.
            onlyif: test
            push: true
        dev:
            name: smuggler
            onlyif: make
            push: false
        ^test.*:
            name: test
            push: false


This file is trying to show what you can do with smg, everything will be detailed in the full documentation

## Tests

Tests are not implemented yet, criticals are expected to reach a good beta status.

## Building system

The build system is tight with git, each of  your image will be built with the following tags:

- Commit (Git)
- Branch (Git) 
- Latest (Docker)
- Tag (Git, if exists for the associated commit)

## Authors

Jean-Baptiste Dalido https://github.com/jbdalido

Vincent Rischmann https://github.com/vrischmann

Nicolas Douillet https://github.com/minimarcel


