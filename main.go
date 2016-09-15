package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	/* Our argument will be a json file that describes
	 * the backup job(s) to run.
	 * That file itself is an encoding of the Job
	 * structure. (backup.go)
	 * TODO : Support:
	 * - Removing old editions?
	 * - Excludes in restore?
	 * - Extra excludes on the command line (useful for restore)?
	 * - Decrypt only (producing the .tar.gz)?
	 * - Log file and stats printed?
	 */
	backup := flag.Bool("backup", false, "Set this to do a backup")
	test := flag.Bool("test", false, "Set this to test the backup files and list their contents")
	restore := flag.Bool("restore", false, "Set this to do a restore")
	jobs := flag.String("job", "backup.json", "Json file describing the backup job")
	prefix := flag.String("prefix", "", "Optional restore prefix")
	replaceStart := flag.String("replaceStart", "", fmt.Sprintf("Optional list of <start of path in archive>%c<replacement>%c...", os.PathListSeparator, os.PathListSeparator))
	replaceAny := flag.String("replace", "", fmt.Sprintf("Optional list of <path in archive>%c<replacement>%c...", os.PathListSeparator, os.PathListSeparator))
	replaceAll := flag.String("replaceAll", "", fmt.Sprintf("Optional list of <path in archive>%c<replacement>%c...", os.PathListSeparator, os.PathListSeparator))

	flag.Parse()

	var err error
	if *backup {
		err = RunBackup(*jobs)
	} else {
		repl := new(Replacements)
		err = repl.AddReplStart(*replaceStart)
		if err != nil {
			fmt.Printf("replaceStart : %s\n", err.Error())
			os.Exit(1)
		}

		err = repl.AddReplAny(*replaceAny)
		if err != nil {
			fmt.Printf("replace : %s\n", err.Error())
			os.Exit(1)
		}

		err = repl.AddReplAll(*replaceAll)
		if err != nil {
			fmt.Printf("replaceAll : %s\n", err.Error())
			os.Exit(1)
		}

		what := -1
		if *restore {
			what = Unpack_Restore
		} else if *test {
			what = Unpack_Test
		} else {
			fmt.Printf("No action specified\n")
			os.Exit(1)
		}

		err = RunUnpack(*jobs, *prefix, repl, what)
	}

	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
