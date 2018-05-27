# codecommit-hugo-s3-lambda

a lambda function that clones a codecommit repo, does a hugo build, and pushes the artifacts to s3

## TODO

a job for the code to run (hugo?)
the s3 delivery
stabilize the build environment (documentation)

## setup notes

### amazon linux docker container

```bash
docker pull amazonlinux
docker run -ti -v ~/go:/root/ amazonlinux
```

### configure the container

just remember that you're working with amazon linux which is its own distribution, and a ton of stuff is missing.

just use yum's go, because that its the quickest way to get all the package dependancies when calling `cmake`, otherwise
its a pain in the butt to get it working.

```bash
#!/bin/bash

# isntall pkg
yum install golang -y
yum install wget -y
yum install curl-devel -y
yum install libssh2-devel -y
yum install http-parser-devel -y
yum install zip -y
yum install cmake -y

# setup go paths
export PATH=$PATH:/usr/local/go/bin
export PATH=$PATH:/root/go/bin/

# make libgit2
wget https://github.com/libgit2/libgit2/archive/v0.26.3.tar.gz
tar xzf v0.26.3.tar.gz
cd libgit2-0.26.3 && mkdir build && cd build
cmake -DCURL=OFF ..
cmake --build .

# link pkgconfig dir for getting git2go.v26
export PKG_CONFIG_PATH=/usr/local/lib/pkgconfig
go get "gopkg.in/libgit2/git2go.v26"

# install deps
mkdir -p /root/go/bin
curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
```

## Resources and Acknowledgements
a working example in python:
https://github.com/0xlen/aws-lambda-python-codecommit-s3-deliver

a great example of aws v4 signing in go, and a working implementation (this guy kicks ass)
https://medium.com/@nzoschke/codecommit-authentication-helper-in-golang-64f0ef42fc61
