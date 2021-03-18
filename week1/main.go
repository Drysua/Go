package main

import (
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

// выводит дерево подкаталогов
func dirTree(out io.Writer, root string, printFiles bool) (err error) {
	// root, _ := os.Getwd()
	// fullPath := filepath.Join(root, "testdata")

	// fmt.Println(fullPath)
	// fullPath, _ := filepath.Abs(filepath.Dir(root))
	// os.Chdir(fullPath)
	size := make([]int, 1)
	k := 1
	filepath.Walk(root, func(path string, info os.FileInfo, err error) error {
		// fmt.Println(size)
		if err != nil {
			fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", path, err)
			return err
		}
		filePath := strings.Split(path, "/")
		// fmt.Println(filePath)
		if !info.IsDir() {

			if printFiles == true {

				for i := 0; i < len(size)-1; i++ {

					if size[i] == 0 {
						fmt.Fprint(out, "	")
					} else {
						fmt.Fprint(out, "│	")
					}
				}

				if size[len(size)-1] > 1 {
					fmt.Fprint(out, "├───"+filePath[len(filePath)-1])
				} else {
					fmt.Fprint(out, "└───"+filePath[len(filePath)-1])
				}

				if info.Size() != 0 {
					fmt.Fprintln(out, " ("+fmt.Sprint(info.Size())+"b)")
				} else {
					fmt.Fprintln(out, " (empty)")
				}
			}

			size[len(size)-k]--
			for i := 0; i <= len(size); i++ {

				if size[len(size)-1] != 0 {
					break
				}
				size = size[:len(size)-1]
				// fmt.Println(size)
			}

		} else {

			if path != root {

				for i := 0; i < len(size)-1; i++ {
					if size[i] == 0 {
						fmt.Fprint(out, "	")
					} else {
						fmt.Fprint(out, "│	")
					}
				}

				if size[len(size)-1] > 1 {
					fmt.Fprintln(out, "├───"+filePath[len(filePath)-1])
				} else {
					fmt.Fprintln(out, "└───"+filePath[len(filePath)-1])
				}
			}

			files, _ := ioutil.ReadDir(path)
			folders := 0
			if !printFiles {
				// fmt.Println(files)
				for _, f := range files {
					if f.IsDir() {
						folders++
					}
				}
			}

			if size[len(size)-k] == 0 {
				if len(size) >= 2 {
					if size[len(size)-2] == 0 {
						size = size[:len(size)-k]
					}
					size[len(size)-2]--
				}
				if !printFiles {
					size[len(size)-k] = folders
				} else {
					size[len(size)-k] = len(files)
				}
			} else {
				size[len(size)-k]--
				if k != 1 {
					if printFiles {
						size[len(size)-k] = folders
					} else {
						size[len(size)-k] = len(files)
					}
				} else {
					size = append(size, len(files))
				}
			}

		}
		return nil
	})
	return
}

func main() {
	out := os.Stdout
	if !(len(os.Args) == 2 || len(os.Args) == 3) {
		panic("usage go run main.go . [-f]")
	}
	path := os.Args[1]
	printFiles := len(os.Args) == 3 && os.Args[2] == "-f"
	err := dirTree(out, path, printFiles)
	if err != nil {
		panic(err.Error())
	}
}
