package main

import (
	"flag"
	"fmt"
	"os"
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
	jobs := flag.String("job", "backup.json", "Json file describing the backup job")
	prefix := flag.String("prefix", "", "Optional path prefix")
	replaceStart := flag.String("replaceStart", "", fmt.Sprintf("Optional list of <start of path in archive>%s<replacement>%s...", sep, sep))
	replaceAny := flag.String("replace", "", fmt.Sprintf("Optional list of <path in archive>%s<replacement>%s...", sep, sep))
	replaceAll := flag.String("replaceAll", "", fmt.Sprintf("Optional list of <path in archive>%s<replacement>%s...", sep, sep))
	include := flag.String("include", "", fmt.Sprintf("Optional list of <path>%s<path>%s... to include", sep, sep))
	exclude := flag.String("exclude", "", fmt.Sprintf("Optional list of <path>%s<path>%s... to exclude", sep, sep))

	flag.Parse()

	includeArray := strings.Split(*include, sep)
	excludeArray := strings.Split(*exclude, sep)

	filter := new(Filters).WithIncludes(includeArray).WithExcludes(excludeArray)

	var err error
	if *backup {
		err = RunBackup(*jobs, filter, *prefix)
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

		err = RunUnpack(*jobs, filter, *prefix, repl, what)
	}

	if err != nil {
		fmt.Printf("%s\n", err.Error())
		os.Exit(1)
	} else {
		os.Exit(0)
	}
}
