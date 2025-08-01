package cmd

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	prompt "github.com/c-bata/go-prompt"
	"github.com/chickiexd/ez_copy/logger"
	"github.com/ktr0731/go-fuzzyfinder"
	"github.com/spf13/cobra"
)

// rootCmd represents the base command when called without any subcommands
var rootCmd = &cobra.Command{
	Use:   "ez_copy",
	Short: "A brief description of your application",
	Long: `A longer description that spans multiple lines and likely contains
examples and usage of using your application. For example:

Cobra is a CLI library for Go that empowers applications.
This application is a tool to generate the needed files
to quickly create a Cobra application.`,
	// Uncomment the following line if your bare application
	// has an action associated with it:
	Run: func(cmd *cobra.Command, args []string) {
		cmd.Println("Welcome to ez_copy!")
		dry_run, _ := cmd.Flags().GetBool("dry-run")
		debug, _ := cmd.Flags().GetBool("debug")
		ez_copy(dry_run, debug)
	},
}

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	// Here you will define your flags and configuration settings.
	// Cobra supports persistent flags, which, if defined here,
	// will be global for your application.

	// rootCmd.PersistentFlags().StringVar(&cfgFile, "config", "", "config file (default is $HOME/.ez_copy.yaml)")

	// Cobra also supports local flags, which will only run
	// when this action is called directly.
	rootCmd.Flags().BoolP("toggle", "t", false, "Help message for toggle")
	rootCmd.Flags().BoolP("dry-run", "n", false, "Perform a dry run without making any changes")
	rootCmd.Flags().BoolP("debug", "d", false, "Enable debug mode")
}

func ez_copy(dry_run bool, debug bool) {
	fmt.Println("Insert download path:")
	download_path := get_path()
	fmt.Println("Download path:", download_path)
	var mkvFiles []string
	err := filepath.WalkDir(download_path, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() && strings.HasSuffix(d.Name(), ".mkv") {
			mkvFiles = append(mkvFiles, path)
		}
		return nil
	})
	if err != nil {
		logger.Log.Errorw("Error reading directory", "error", err)
		return
	}
	if len(mkvFiles) == 0 {
		fmt.Println("No .mkv files found in the specified directory.")
		logger.Log.Infow("No .mkv files found", "path", download_path)
		return
	}
	selectedFile, err := search_mkv_files(mkvFiles)
	if err != nil {
		logger.Log.Errorw("Error searching for .mkv files", "error", err)
		return
	}
	fmt.Println("Selected file:", selectedFile)
	filter := prompt.Input("Filter files for string: ", completer)
	filteredFiles := []string{}
	if filter != "" {
		for _, file := range mkvFiles {
			if strings.Contains(filepath.Base(file), filter) && !strings.Contains(filepath.Base(file), "sample") {
				filteredFiles = append(filteredFiles, file)
			}
		}
		if len(filteredFiles) == 0 {
			fmt.Println("No files found with the specified filter.")
			logger.Log.Infow("No files found with filter", "filter", filter)
			return
		}
		fmt.Printf("Found %d .mkv files for filter '%s'.\n", len(filteredFiles), filter)
	}

	fmt.Println("Insert destination path:")
	destination_path := get_path()
	fmt.Println("Destination path:", destination_path)
	dirName := prompt.Input("Insert destination directory name: ", completer)
	if dirName == "" {
		prompt_text := "No directory name provided, using destination path as directory name."
		input := yes_no_prompt(prompt_text)
		if !input {
			fmt.Println("Exiting without copying.")
			logger.Log.Infow("No directory name provided, exiting", "destination_path", destination_path)
			return
		}
	} else {
		destination_path = filepath.Join(destination_path, dirName)
	}
	confirmation := fmt.Sprintf("You are about to copy the files with filter: '%s' to the destination: '%s'.", filter, destination_path)
	input := yes_no_prompt(confirmation)
	if !input {
		fmt.Println("Exiting without copying.")
		logger.Log.Infow("User cancelled the operation", "confirmation", confirmation)
		return
	}
	if debug {
		logger.Log.Infow("Starting copy operation", "filter", filter, "destination", destination_path, "dry_run", dry_run)
	}
	fmt.Println("Copying files...")
	err = copy_files(filteredFiles, destination_path, dry_run)
	fmt.Println("Copy operation completed.")
	input = yes_no_prompt("Do you want to delete the source files?")
	if input {
		// TODO
		fmt.Println("Deleting source files...")
		fmt.Println("Source files deleted.")
	} else {
		fmt.Println("Source files not deleted.")
	}
}

func copy_files(files []string, destination string, dry_run bool) error {
	for _, file := range files {
		destFile := filepath.Join(destination, filepath.Base(file))
		if dry_run {
			fmt.Printf("[DRY RUN] Copying %s to %s\n", file, destFile)
		} else {
			err := os.MkdirAll(filepath.Dir(destFile), os.ModePerm)
			if err != nil {
				logger.Log.Errorw("Error creating directory", "error", err, "directory", filepath.Dir(destFile))
				continue
			}

			sourceFile, err := os.Open(file)
			if err != nil {
				logger.Log.Errorw("Error opening source file", "error", err, "file", file)
				return err
			}
			defer sourceFile.Close()
			destinationFile, err := os.Create(destFile)
			if err != nil {
				logger.Log.Errorw("Error creating destination file", "error", err, "file", destFile)
				return err
			}
			defer destinationFile.Close()
			_, err = io.Copy(destinationFile, sourceFile)
			if err != nil {
				logger.Log.Errorw("Error copying file", "error", err, "source", file, "destination", destFile)
				return err
			}
			err = destinationFile.Sync()
			if err != nil {
				logger.Log.Errorw("Error syncing destination file", "error", err, "file", destFile)
				return err
			}
			fmt.Printf("Copied %s to %s\n", file, destFile)
		}
	}
	return nil
}

func yes_no_prompt(question string) bool {
	fmt.Println(question + " (yes/no)")
	for {
		input := prompt.Input("> ", y_n_completer)
		input = strings.TrimSpace(strings.ToLower(input))
		switch input {
		case "y", "yes":
			return true
		case "n", "no":
			return false
		default:
			fmt.Println("Please enter 'yes' or 'no'.")
		}
	}
}

func y_n_completer(d prompt.Document) []prompt.Suggest {
	return []prompt.Suggest{
		{Text: "yes", Description: "Answer yes"},
		{Text: "no", Description: "Answer no"},
	}
}

func search_mkv_files(items []string) (string, error) {
	idx, err := fuzzyfinder.Find(
		items,
		func(i int) string {
			return items[i]
		},
		fuzzyfinder.WithPromptString("Search> "),
	)
	if err != nil {
		return "", err
	}
	return items[idx], nil
}

func get_path() string {
	fullPath := prompt.Input("> ", completer,
		prompt.OptionTitle("path input"),
		prompt.OptionPrefixTextColor(prompt.Yellow),
		prompt.OptionAddKeyBind(prompt.KeyBind{
			Key: prompt.ControlC,
			Fn: func(buf *prompt.Buffer) {
				fmt.Println("\n ctrl+c detected, exiting...")
				os.Exit(1)
			},
		}),
	)
	return fullPath
}

func completer(d prompt.Document) []prompt.Suggest {
	text := d.TextBeforeCursor()
	text = filepath.Clean(text)
	if len(text) <= 2 && !strings.Contains(text, string(os.PathSeparator)) {
		return suggestDrives()
	}
	dir := text
	info, err := os.Stat(dir)
	if err != nil || !info.IsDir() {
		dir = filepath.Dir(text)
	}
	files, err := os.ReadDir(dir)
	if err != nil {
		return nil
	}
	base := filepath.Base(text)
	suggestions := []prompt.Suggest{}
	for _, file := range files {
		name := file.Name()
		if strings.HasPrefix(strings.ToLower(name), strings.ToLower(base)) {
			suffix := ""
			if file.IsDir() {
				suffix = string(os.PathSeparator)
			}
			suggestions = append(suggestions, prompt.Suggest{
				Text: filepath.Join(dir, name) + suffix,
			})
		}
	}
	return suggestions
}

func suggestDrives() []prompt.Suggest {
	drives := []prompt.Suggest{}
	for c := 'A'; c <= 'Z'; c++ {
		drive := fmt.Sprintf("%c:\\", c)
		if _, err := os.Stat(drive); err == nil {
			drives = append(drives, prompt.Suggest{
				Text: drive,
			})
		}
	}
	return drives
}
