xml-image-dmp
========
A tiny tool to export base64-encoded images in an xml (not from html)

Usage
-----

    $ xml-image-dmp [-i input_file[,another_file]] [-x xpath] [-n name_to_save] [-d attribute_name_of_data]

### Options
`-i` :: You can specify input files. If you want to specify multiple files, call this option multiple times.
You can also specify multiple files by seprating with comma ",".
If this option is not specified, all xml files in the current directory will be processed.

`-x` :: An XPath expression to locate image resources in an xml. Selectin attribute values is not supported.

`-n` :: This tool assumes there is an attribute that corresponds to the filename. You can specify the attribute name
of the image. The attribute must be in the same tag that is specified by `-x` option.

`-d` :: This option set the attribute name of data from which this tool extract an image.
