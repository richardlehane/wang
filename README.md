# Wang

This code inspects and carves files from disks formatted as a Wang "Archive Diskette" (see Chapter 8: [https://www.wang2200.org/docs/software/2200WordProcessingOperatorsGuide.700-6937.6-82.pdf](https://www.wang2200.org/docs/software/2200WordProcessingOperatorsGuide.700-6937.6-82.pdf)).

## Install

Install go: [https://go.dev/doc/install](https://go.dev/doc/install)

Run `go install github.com/richardlehane/wang/cmd/wang@latest`

## Usage

    wang meta DISK.IMG    // Provides a directory listing for DISK.IMG
    wang files DISK.IMG   // Extracts files into the working directory
    wang dump DISK.IMG    // Dumps sectors with tagged content into the working directory 
