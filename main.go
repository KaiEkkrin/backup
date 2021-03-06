package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	sep := fmt.Sprintf("%c", os.PathListSeparator)

	/* Our argument will be a json file that describes
	 * the backup job(s) to run.
	 * That file itself is an encoding of the Job
	 * structure. (backup.go)
	 * TODO : Support:
	 * - Removing old editions?
	 * - Decrypt only (producing the .tar.gz)?
	 * - Log file and stats printed?
	 */
	backup := flag.Bool("backup", false, "Set this to do a backup")
	test := flag.Bool("test", false, "Set this to test the backup files and list their contents")
	restore := flag.Bool("restore", false, "Set this to do a restore")
	listEditions := flag.Bool("listEditions", false, "Set this to just list the editions of this backup")

	jobs := flag.String("job", "backup.json", "Json file describing the backup job")
	prefix := flag.String("prefix", "", "Optional path prefix")
	replaceStart := flag.String("replaceStart", "", fmt.Sprintf("Optional list of <start of path in archive>%s<replacement>%s...", sep, sep))
	replaceAny := flag.String("replace", "", fmt.Sprintf("Optional list of <path in archive>%s<replacement>%s...", sep, sep))
	replaceAll := flag.String("replaceAll", "", fmt.Sprintf("Optional list of <path in archive>%s<replacement>%s...", sep, sep))
	include := flag.String("include", "", fmt.Sprintf("Optional list of <path>%s<path>%s... to include", sep, sep))
	exclude := flag.String("exclude", "", fmt.Sprintf("Optional list of <path>%s<path>%s... to exclude", sep, sep))
	removeAfter := flag.String("removeAfter", "", fmt.Sprintf("Optional edition to base the backup on"))

	flag.Parse()

	includeArray := strings.Split(*include, sep)
	excludeArray := strings.Split(*exclude, sep)

	filter := new(Filters).WithIncludes(includeArray).WithExcludes(excludeArray)

	// Change into the directory of the job spec:
	oldWd, err := os.Getwd()
	if err != nil {
		fmt.Printf("Getwd : %s\n", err.Error())
		os.Exit(1)
	}
	defer os.Chdir(oldWd)

	jobDir, jobFile := filepath.Split(*jobs)
	if len(jobDir) > 0 {
		err = os.Chdir(jobDir)
		if err != nil {
			fmt.Printf("Chdir %s : %s\n", jobDir, err.Error())
			os.Exit(1)
		}
	}

	if *backup {
		var removeAfterEdition *Edition
		if len(*removeAfter) > 0 {
			removeAfterEdition, err = EditionFromString(*removeAfter)
			if err != nil {
				fmt.Printf("removeAfter : %s\n", err.Error())
				os.Exit(1)
			}
		}

		err = RunBackup(jobFile, filter, *prefix, removeAfterEdition)
	} else if *listEditions {
		err = RunListEditions(jobFile)
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

		err = RunUnpack(jobFile, filter, *prefix, repl, what)
	}

	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
