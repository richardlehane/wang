# Wang

This code inspects and carves files from disks formatted as a Wang "Archive Diskette" (see Chapter 8: [https://www.wang2200.org/docs/software/2200WordProcessingOperatorsGuide.700-6937.6-82.pdf](https://www.wang2200.org/docs/software/2200WordProcessingOperatorsGuide.700-6937.6-82.pdf)).

Files are further processed to convert to RTF format (see *File Format Documentation for UN Parallel Texts* for more information about the Wang WP character set: [https://catalog.ldc.upenn.edu/docs/LDC94T4B-2/wang2iso.txt](https://catalog.ldc.upenn.edu/docs/LDC94T4B-2/wang2iso.txt)). *Both cited references are also in the /docs folder*.

## Install

Install go: [https://go.dev/doc/install](https://go.dev/doc/install)

Run `go install github.com/richardlehane/wang/cmd/wang@latest`

## Usage

    wang meta DISK.IMG    // Provides a directory listing and metadata for DISK.IMG
    wang csv DISK.IMG     // Directory listing and metadata in CSV format
    wang files DISK.IMG   // Extracts files into the working directory
    wang text DISK.IMG    // Files are converted to text and extracted to working directory
    wang rtf DISK.IMG     // Files are converted to rtf and extracted to working directory
    wang fix DISK.IMG     // Attempts to fix a broken wang image by rewriting the image to the working directory
    wang dump DISK.IMG    // Dumps sectors with tagged content into the working directory
