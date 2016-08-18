package main

import (
	"encoding/base64"
	"encoding/xml"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/beevik/etree"
)

// structs used to parse XML files
type xmlCatalogue struct {
	XMLName xml.Name  `xml:"catalog"`
	Img     []*xmlImg `xml:"book>img"`
}

type xmlImg struct {
	XMLName    xml.Name `xml:"img"`
	FileName   string   `xml:"filename,attr"`
	BinaryData string   `xml:"bin,attr"`
}

// alias of []string to parse command line arguments
type filenames []string

func process(filename string) {
	log.Printf("Processing %s ...\n", filename)
	f, err := os.Open(filename)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	// read all bytes from the reader `f`
	data, err := ioutil.ReadAll(f)
	if err != nil {
		log.Fatal(err)
	}

	// declare a data which stores the xml data
	catalogue := new(xmlCatalogue)
	// Convert xml byte data into Go struct
	if err := xml.Unmarshal(data, &catalogue); err != nil {
		log.Fatal(err)
	}

	for _, img := range catalogue.Img {
		base := createTargetDir(filename)

		log.Printf("Found image: %s\n", img.FileName)
		data := excludeScheme(img.BinaryData)
		decoded, err := base64.StdEncoding.DecodeString(data)
		if err != nil {
			log.Println("Invalid base64 encoding. Cannot be decoded.")
			log.Println(err)
			dumpErrorImg(decoded, 30, 2)
			continue
		}

		err = writeImage(base+"/"+img.FileName, decoded)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func createTargetDir(filename string) string {
	base := extractBaseName(filename)
	if !dirExists(base) {
		log.Printf("Creating a directory to store images : %s\n", base)
		err := os.Mkdir(base, os.FileMode(0766))
		if err != nil {
			log.Fatal(err)
		}
	}

	return base
}

func writeImage(filepath string, data []byte) error {
	log.Printf("Writing the image to %s\n", filepath)
	fo, err := os.Create(filepath)
	if err != nil {
		return err
	}
	defer fo.Close()

	wrote, err := fo.Write(data)
	if err != nil || wrote != len(data) {
		log.Printf("Something wrong while writing an image to a file: %s\n", filepath)
		return err
	}
	return nil
}

// processWithXPath receives 2 args, a filename and an XPath,
// then tries to parse the file with the XPath
// The XPath should specify the node which contains Base64-encoded
// image resources.
func processWithXPath(filename, nodeXpath, imgAttr, nameAttr string) {
	doc := etree.NewDocument()
	if err := doc.ReadFromFile(filename); err != nil {
		log.Fatal(err)
	}

	for i, node := range doc.FindElements(nodeXpath) {
		log.Printf("Processing node %0d : %s", i, node.Tag)

		// retreive image data. If not found, skip this iteration
		img := node.SelectAttrValue(imgAttr, "")
		if img == "" {
			log.Printf("Node %s has been specified, but does not contains an image.\n", node.Tag)
			continue
		}
		imgdata := excludeScheme(img)

		// retreive name. If not found, give the image an temporary name
		name := node.SelectAttrValue(nameAttr, fmt.Sprintf("img_%04d", i))

		dirname := createTargetDir(filename)

		decoded, err := base64.StdEncoding.DecodeString(imgdata)
		if err != nil {
			log.Println("Invalid base64 encoding. Cannot be decoded.")
			log.Println(err)
			dumpErrorImg(decoded, 30, 2)
			continue
		}

		err = writeImage(dirname+"/"+name, decoded)
		if err != nil {
			log.Fatal(err)
		}
	}
}

func dumpErrorImg(data []byte, bytesInRow int, rowLim int) {
	dataLen := len(data)
	log.Printf("Decoded data: %d bytes\n", dataLen)
	for r := 0; r < rowLim; r++ {
		start := r * bytesInRow
		lim := (r + 1) * bytesInRow
		if lim > dataLen {
			lim = dataLen
		}
		log.Printf("%X\n", data[start:lim])
	}
}

func extractBaseName(filename string) string {
	base := path.Base(filename)
	lastIndex := strings.LastIndex(base, ".")
	return base[:lastIndex]
}

func dirExists(filename string) bool {
	stat, err := os.Stat(filename)
	return err == nil && stat.IsDir()
}

// String : a method of the flag.Value interface
// Shows all filenames by joining them with a comma and a space
func (fs *filenames) String() string {
	return strings.Join([]string(*fs), ", ")
}

func (fs *filenames) Set(value string) error {
	for _, filename := range strings.Split(value, ",") {
		*fs = append(*fs, filename)
	}
	return nil
}

func listFilenames() (filenames, error) {
	var filteredFilenames filenames
	rawFilenames, err := ioutil.ReadDir(".")
	if err != nil {
		return filteredFilenames, err
	}

	for _, f := range rawFilenames {
		if strings.HasSuffix(f.Name(), ".xml") {
			filteredFilenames = append(filteredFilenames, f.Name())
		}
	}

	return filteredFilenames, nil
}

func excludeScheme(data string) string {
	// http://www.ietf.org/rfc/rfc2397.txt
	r := regexp.MustCompile("^data:(?:\\w+\\/\\w+)(?:;(?:\\w+=\\w+)+)?;base64,")
	locs := r.FindStringIndex(data)
	if locs == nil {
		return data
	}

	return data[locs[1]:]
}

// Read an XML and extract images which is encoded by base64,
// then write the images to a folder.
func main() {
	// parse command line arguments
	var inputFiles filenames
	flag.Var(&inputFiles, "i",
		"You can specify multiple files by setting this flag multiple times or write file names with separation of a comma",
	)

	var xpath string
	flag.StringVar(&xpath, "x", ".//img",
		"XPath string to locate tags that contain images",
	)

	var data string
	flag.StringVar(&data, "d", "bin",
		"Name of the attribute which contains actual data within the node. This tool assumes the data is inside an attribute.",
	)

	var filename string
	flag.StringVar(&filename, "n", "filename",
		"Name of the attribute which specifies the name of image data",
	)

	flag.Parse()

	// check the length of filenames. If no files specified, all XML files
	// should be the target of the tool
	if len(inputFiles) <= 0 {
		log.Println("No files specified. Parse all XML files in the current directory")
		inputFiles, _ = listFilenames()
	}

	for i, f := range inputFiles {
		log.Printf("Processing file (No. %d, Name %s)", i, f)
		//process(f)
		processWithXPath(f, xpath, data, filename)
	}

}
