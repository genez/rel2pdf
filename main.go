package main

import (
	"bufio"
	"fmt"
	"github.com/jung-kurt/gofpdf"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Header struct {
	ProtocolloTelematico string
	TipoRecord           string
}

type Record interface {
}

type RecordP struct {
	Header

	CodiceFornitura         string
	NomeFileCorto           string
	DataRicezione           time.Time
	TotaleDocumentiAccolti  int
	TotaleDocumentiRespinti int
	Versione                int
	NomeFileLungo           string
	TotaleRecordRicevuta    int
	Titolo                  string
	TabellaTesto            []RigaTesto
}

type RigaTesto struct {
	TipoRiga string
	Testo    string
}

type RecordRQ struct {
	Header
	PrimaPagina bool

	ProgressivoProtocollo   int
	NomeFileCorto           string
	ProgressivoRicevuta     int
	TotaleRecordRicevuta    int
	CodiceFiscalePartitaIva string
	Denominazione           string
	SaltoPagina             bool
	TabellaTesto            []RigaTesto
}

func main() {

	relFileName := os.Args[1]

	pdfFileName := strings.TrimSuffix(relFileName, filepath.Ext(relFileName)) + ".pdf"

	f, err := os.Open(relFileName)
	if err != nil {
		log.Fatal(err)
	}
	defer f.Close()

	records := make([]Record, 0)

	pdf := gofpdf.New("P", "mm", "A4", "")
	pdf.SetFont("Courier", "B", 12)
	pdf.SetHeaderFunc(func() {
		pdf.SetFont("Courier", "B", 12)
		pdf.ImageOptions("logo.png", 85, 5, 0, 0, false, gofpdf.ImageOptions{}, 0, "")
		pdf.SetY(20)
		pdf.SetFont("Courier", "B", 8)
		pdf.MultiCell(0, 6,
			"SERVIZIO TELEMATICO ENTRATEL DI PRESENTAZIONE DELLE DICHIARAZIONI\nCOMUNICAZIONE DI AVVENUTO RICEVIMENTO (art. 3, comma 10, D.P.R. 322/1998)",
			"",
			"C",
			false)
	})
	pdf.SetFooterFunc(func() {
		pdf.SetFont("Courier", "B", 12)
		pdf.SetY(-15)
		pdf.CellFormat(0, 10, fmt.Sprintf("%d", pdf.PageNo()),
			"", 0, "C", false, 0, "")
	})
	//pdf.AliasNbPages("")

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()

		h := Header{}
		h.ProtocolloTelematico = line[:17]
		h.TipoRecord = line[17 : 17+1]

		switch h.TipoRecord {
		case "P":
			p := &RecordP{Header: h}
			p.CodiceFornitura = line[27 : 27+5]
			p.NomeFileCorto = strings.TrimSpace(line[38 : 38+8])
			p.DataRicezione, _ = time.Parse("20060102", line[46:46+8])
			p.TotaleDocumentiAccolti, _ = strconv.Atoi(line[54 : 54+6])
			p.TotaleDocumentiRespinti, _ = strconv.Atoi(line[60 : 60+6])
			p.NomeFileLungo = strings.TrimSpace(line[68 : 68+47])
			p.Versione, _ = strconv.Atoi(line[115 : 115+6])
			p.TotaleRecordRicevuta, _ = strconv.Atoi(line[121 : 121+6])
			p.Titolo = strings.TrimSpace(line[250 : 250+150])
			p.TabellaTesto = make([]RigaTesto, 0)
			for i := 0; i < 20; i++ {
				offset := 400 + (i * 80)
				rt := RigaTesto{
					TipoRiga: line[offset : offset+1],
					Testo:    strings.TrimSpace(line[offset+1 : offset+79]),
				}
				if rt.TipoRiga == "F" {
					break
				}
				p.TabellaTesto = append(p.TabellaTesto, rt)
			}
			fmt.Printf("%#v\n", p)
			records = append(records, p)
			break
		case "R":
		case "Q":
			rq := &RecordRQ{Header: h}
			rq.ProgressivoProtocollo, _ = strconv.Atoi(line[18 : 18+9])
			rq.NomeFileCorto = strings.TrimSpace(line[38 : 38+8])
			rq.TabellaTesto = make([]RigaTesto, 0)
			for i := 0; i < 19; i++ {
				offset := 480 + (i * 80)
				rt := RigaTesto{
					TipoRiga: line[offset : offset+1],
					Testo:    strings.TrimSpace(line[offset+1 : offset+79]),
				}
				if rt.TipoRiga == "F" {
					break
				} else {
					if rt.TipoRiga == "P" {
						rq.PrimaPagina = true
					}
				}
				rq.TabellaTesto = append(rq.TabellaTesto, rt)
			}
			records = append(records, rq)
			break
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatal(err)
	}

	for _, rec := range records {
		if p, ok := rec.(*RecordP); ok {
			startDocument(pdf, p)
		}
	}

	for _, rec := range records {
		rq, ok := rec.(*RecordRQ)
		if !ok {
			continue
		}

		if !rq.PrimaPagina {
			continue
		}

		addInFirstPage(pdf, rq)
	}

	for _, rec := range records {
		if rq, ok := rec.(*RecordRQ); ok {
			addPage(pdf, rq)
		}
	}

	err = pdf.OutputFileAndClose(pdfFileName)
	if err != nil {
		log.Fatal(err)
	}
}

func addInFirstPage(f *gofpdf.Fpdf, rq *RecordRQ) {
	f.SetFont("Courier", "", 8)
	f.CellFormat(0, 6, rq.TabellaTesto[0].Testo, "", 2, "C", false, 0, "")
}

func addPage(f *gofpdf.Fpdf, rq *RecordRQ) {
	f.SetFont("Courier", "", 8)
	f.AddPage()
}

func startDocument(f *gofpdf.Fpdf, p *RecordP) {
	f.SetFont("Courier", "", 8)
	f.AddPage()

	f.Ln(-1)

	f.CellFormat(0, 6, p.Titolo, "", 2, "C", false, 0, "")

	for _, c := range p.TabellaTesto {
		f.CellFormat(0, 5, c.Testo, "", 2, "", false, 0, "")
	}
	f.Ln(-1)
	f.CellFormat(0, 5, "Li, "+p.DataRicezione.Format("02/01/2006"), "", 2, "", false, 0, "")
	f.Ln(-1)

}
