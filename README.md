# NFS Exporter

NFS Exporter is a tool for exporting NFS metrics to a prom format, then it can be easily integrated to Prometheus using `node_exporter` textfile collector.

## Features

- Collects and exports NFS metrics
- Supports multiple architectures
- Easy to integrate with Prometheus

## Installation

You can download the latest release from the [releases page](https://github.com/wy0917/nfs_exporter/releases).

## Usage

After downloading, you can run the NFS Exporter with the following command:

```bash
./nfs_exporter
```

You can control the behavior of the NFS Exporter with the following flags:

- `-o`: Specify the output file path. __Default is standard output__.
- `-f`: Specify the name of the test file. Default is `.testfile`.
- `-t`: Specify the timeout in milliseconds. Default is 200.
- `-V`: Enable verbose mode to print debug information. Default is false.

## Building from Source

If you want to build the NFS Exporter from source, you can do so with the following commands:

```bash
git clone https://github.com/wy0917/nfs_exporter.git
cd nfs_exporter
GOOS=linux GOARCH=amd64 go build -o nfs_exporter
```

This will create a binary called `nfs_exporter` in the current directory.

## License

This project is licensed under the GPLv3 License - see the [GPLv3 License](https://www.gnu.org/licenses/gpl-3.0.en.html) for details.
