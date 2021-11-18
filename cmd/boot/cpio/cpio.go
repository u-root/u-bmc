package main

import (
	"flag"
	"log"
	"os"
	"path/filepath"

	"github.com/u-root/u-root/pkg/cpio"
	"github.com/u-root/u-root/pkg/uroot/initramfs"
)

var (
	outPath = flag.String("out", "", "Path to output initramfs")
)

func main() {
	flag.Parse()
	logger := log.New(os.Stderr, "", log.LstdFlags)

	archiver, err := initramfs.GetArchiver("cpio")
	if err != nil {
		log.Fatalf("GetArchiver(cpio): %v", err)
	}
	writer, err := archiver.OpenWriter(logger, *outPath)
	if err != nil {
		log.Fatalf("Openwriter(%s): %v", *outPath, err)
	}

	files := initramfs.NewFiles()
	args := flag.Args()
	if len(args) != 0 {
		for _, arg := range args {
			extraFiles, _ := filepath.Glob(arg)
			if extraFiles == nil {
				log.Fatalf("Glob(%s): no such file", arg)
			}
			for _, f := range extraFiles {
				err := files.AddFile(f, filepath.Base(f))
				if err != nil {
					log.Fatalf("AddFile(%s): %v", f, err)
				}
			}
		}
	}

	records := []cpio.Record{
		cpio.Directory("ro", 0755),
		cpio.Directory("mnt", 0755),
		cpio.Directory("tmp", 0755),
		cpio.Directory("proc", 0555),
		cpio.Directory("sys", 0555),
		cpio.Directory("dev", 0777),
		cpio.CharDev("dev/console", 0600, 5, 1),
		cpio.CharDev("dev/tty", 0666, 5, 0),
		cpio.CharDev("dev/null", 0666, 1, 3),
		cpio.CharDev("dev/port", 0640, 1, 4),
		cpio.CharDev("dev/urandom", 0666, 1, 9),
	}
	cpio.MakeAllReproducible(records)
	archive := cpio.ArchiveFromRecords(records).Reader()

	opts := &initramfs.Opts{
		Files:           files,
		BaseArchive:     archive,
		OutputFile:      writer,
		UseExistingInit: false,
	}

	err = initramfs.Write(opts)
	if err != nil {
		log.Fatalf("Write(initramfs): %v", err)
	}
}
