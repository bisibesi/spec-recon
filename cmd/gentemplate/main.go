package main

import (
	"archive/zip"
	"os"
)

func main() {
	f, err := os.Create("template.docx")
	if err != nil {
		panic(err)
	}
	defer f.Close()

	w := zip.NewWriter(f)

	// 1. [Content_Types].xml
	ct, _ := w.Create("[Content_Types].xml")
	ct.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Types xmlns="http://schemas.openxmlformats.org/package/2006/content-types">
<Default Extension="rels" ContentType="application/vnd.openxmlformats-package.relationships+xml"/>
<Default Extension="xml" ContentType="application/xml"/>
<Override PartName="/word/document.xml" ContentType="application/vnd.openxmlformats-officedocument.wordprocessingml.document.main+xml"/>
</Types>`))

	// 2. _rels/.rels
	rels, _ := w.Create("_rels/.rels")
	rels.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
<Relationship Id="rId1" Type="http://schemas.openxmlformats.org/officeDocument/2006/relationships/officeDocument" Target="word/document.xml"/>
</Relationships>`))

	// 3. word/_rels/document.xml.rels (Required by some parsers)
	docRels, _ := w.Create("word/_rels/document.xml.rels")
	docRels.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<Relationships xmlns="http://schemas.openxmlformats.org/package/2006/relationships">
</Relationships>`))

	// 4. word/document.xml (Minimal with placeholders)
	doc, _ := w.Create("word/document.xml")
	doc.Write([]byte(`<?xml version="1.0" encoding="UTF-8" standalone="yes"?>
<w:document xmlns:w="http://schemas.openxmlformats.org/wordprocessingml/2006/main">
<w:body>
<w:p><w:r><w:t>Spec Recon Report</w:t></w:r></w:p>
<w:p><w:r><w:t>Date: {{Date}}</w:t></w:r></w:p>
<w:p><w:r><w:t>Total Controllers: {{TotalControllers}}</w:t></w:r></w:p>
<w:p><w:r><w:t>{{Content}}</w:t></w:r></w:p>
</w:body>
</w:document>`))

	w.Close()
}
