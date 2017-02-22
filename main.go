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
	PrimaPagina             bool
	ProgressivoProtocollo   string
	ProgressivoRicevuta     string
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
		case "R", "Q":
			rq := &RecordRQ{Header: h}
			rq.ProgressivoProtocollo = line[18 : 18+9]
			rq.CodiceFiscalePartitaIva = strings.TrimSpace(line[38 : 38+16])
			rq.Denominazione = strings.TrimSpace(line[54 : 54+60])
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

	for _, rec := range records {
		if p, ok := rec.(*RecordP); ok {
			addLastPage(pdf, p)
		}
		if rq, ok := rec.(*RecordRQ); ok {
			addInLastPageDetails(pdf, rq)
		}
	}

	err = pdf.OutputFileAndClose(pdfFileName)
	if err != nil {
		log.Fatal(err)
	}
}

func addInLastPageDetails(f *gofpdf.Fpdf, rq *RecordRQ) {
	f.SetFont("Courier", "", 8)
	f.Ln(-1)

	var esito string
	if rq.TipoRecord == "R" {
		esito = "acquisito"
	} else {
		esito = "scartato"
	}

	f.CellFormat(20, 5, esito, "", 0, "", false, 0, "")
	f.CellFormat(40, 5, rq.ProgressivoProtocollo, "", 0, "", false, 0, "")
	f.CellFormat(40, 5, rq.CodiceFiscalePartitaIva, "", 0, "", false, 0, "")
	f.CellFormat(60, 5, rq.Denominazione, "", 0, "", false, 0, "")
}

func addLastPage(f *gofpdf.Fpdf, p *RecordP) {
	f.SetFont("Courier", "", 8)
	f.AddPage()
	f.Ln(-1)

	f.CellFormat(0, 6, "ELENCO DEI DOCUMENTI ACQUISITI E/O SCARTATI", "", 2, "C", false, 0, "")
	f.Ln(-1)

	f.CellFormat(50, 5, "PROTOCOLLO DI RICEZIONE:", "", 0, "L", false, 0, "")
	f.CellFormat(50, 5, p.ProtocolloTelematico, "", 0, "", false, 0, "")
	f.Ln(-1)
	f.CellFormat(50, 5, "NOME DEL FILE:", "", 0, "L", false, 0, "")
	f.CellFormat(50, 5, p.NomeFileLungo, "", 0, "", false, 0, "")
	f.Ln(-1)
	f.CellFormat(50, 5, "TIPO DOCUMENTO:", "", 0, "L", false, 0, "")
	if p.CodiceFornitura == "I24A0" {
		f.CellFormat(50, 5, "Esito versamento F24", "", 0, "", false, 0, "")
	}
	f.Ln(-1)
	f.CellFormat(50, 5, "DOCUMENTI ACQUISITI:", "", 0, "L", false, 0, "")
	f.CellFormat(10, 5, strconv.Itoa(p.TotaleDocumentiAccolti), "", 0, "R", false, 0, "")
	f.Ln(-1)
	f.CellFormat(50, 5, "DOCUMENTI SCARTATI:", "", 0, "L", false, 0, "")
	f.CellFormat(10, 5, strconv.Itoa(p.TotaleDocumentiRespinti), "", 0, "R", false, 0, "")

	f.Ln(-1)
	f.Ln(-1)
	f.CellFormat(20, 5, "Esito", "", 0, "L", false, 0, "")
	f.CellFormat(40, 5, "Protocollo Documenti", "", 0, "L", false, 0, "")
	f.CellFormat(40, 5, "Codice Fiscale", "", 0, "L", false, 0, "")
	f.CellFormat(60, 5, "Denominazione", "", 0, "L", false, 0, "")
}

func addInFirstPage(f *gofpdf.Fpdf, rq *RecordRQ) {
	f.SetFont("Courier", "", 8)
	f.Ln(-1)
	for _, t := range rq.TabellaTesto {
		if t.TipoRiga == "T" {
			log.Fatal(t)
		}

		if t.TipoRiga == "P" {
			f.CellFormat(0, 6, t.Testo, "", 2, "", false, 0, "")
		}
	}
}

func addPage(f *gofpdf.Fpdf, rq *RecordRQ) {
	f.SetFont("Courier", "", 8)
	f.AddPage()
	f.Ln(-1)
	for _, t := range rq.TabellaTesto {
		if t.TipoRiga == "T" {
			log.Fatal(t)
		}

		f.CellFormat(0, 6, t.Testo, "", 2, "", false, 0, "")
	}
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
