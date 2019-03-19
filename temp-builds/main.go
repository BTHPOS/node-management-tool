package main

import (
	"fmt"

	"github.com/sqweek/dialog"
)

func main() {
	/* Note that spawning a dialog from a non-graphical app like this doesn't
	** quite work properly in OSX. The dialog appears fine, and mouse
	** interaction works but keypresses go straight through the dialog.
	** I'm guessing it has something to do with not having a main loop? */
	dialog.Message("%s", "Please select a folder").Title("Hello world!").Info()
	directory, err := dialog.Directory().Title("Load images").Browse()
	fmt.Println(directory)
	fmt.Println("Error:", err)
	dialog.Message("You chose folder: %s", directory).Title("Goodbye world!").Error()
}
