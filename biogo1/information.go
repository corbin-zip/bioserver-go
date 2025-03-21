package main

import (
	"log"
	"os"
	"strings"
)

type Information struct {
	defaultContents []byte
}

func NewInformation() *Information {
	return &Information{
		defaultContents: []byte(`<HTML>
<HEAD>
<!--
	<GAME-STYLE>
		"MOUSE=OFF",
		"SCROLL=OFF",
		"TITLE=OFF",
		"BACK=ON:mmbb://BUTTON_NG",
		"FORWARD=OFF",
		"CANCEL=OFF",
		"RELOAD=OFF",
		"CHOICE_MV=OFF",
		"X_SHOW=OFF",
		"FRONT_LABEL=ON:6",
	</GAME-STYLE>
-->
<TITLE>database</TITLE><meta http-equiv="Content-Type" content="text/html; charset=Shift_JIS"></HEAD>

<BODY bgcolor="#000033" text=#FFFFFF>
<!-- Choices -->
<br>
<IMG SRC="" width=0 height=0 USEMAP=#CENTER_MAP BORDER=0>
<MAP NAME=CENTER_MAP>
<!--CHG-IMG-BUTTON-2--><AREA SHAPE=RECT COORDS="164, 30,416, 60" HREF=lbs://lbs/02/INFOR/INFOR00.HTM>
<!--CHG-IMG-BUTTON-2--><AREA SHAPE=RECT COORDS="164, 92,416,118" HREF=lbs://lbs/02/RANKING.HTM>
<!--CHG-IMG-BUTTON-2--><AREA SHAPE=RECT COORDS="164,154,416,219" HREF=afs://02/2>
<!--CHG-IMG-BUTTON-2--><AREA SHAPE=RECT COORDS="164,216,416,266" HREF=afs://02/4>
</MAP> 

<table width=584 cellspacing=30 cellpadding=0>
  <tr>
    <td align=center>&nbsp;</td>
    <td width=256 height=32 align=center background=afs://02/123.PNG>INFORMATION</td>
    <td align=center>&nbsp;</td>
  </tr>
  <tr>
    <td align=center>&nbsp;</td>
    <td width=256 height=32 align=center background=afs://02/123.PNG>RANKING</td>
    <td align=center>&nbsp;</td>
  </tr>
  <tr>
    <td align=center>&nbsp;</td>
    <td width=256 height=32 align=center background=afs://02/123.PNG>TERMS OF USE</td>
    <td align=center>&nbsp;</td>
  </tr>
  <tr>
    <td align=center>&nbsp;</td>
    <td width=256 height=32 align=center background=afs://02/123.PNG>REGISTER / CHANGE</td>
    <td align=center>&nbsp;</td>
  </tr>
</table>
</BODY>
</HTML>`),
	}
}

func (inf *Information) GetData(url string) []byte {
	// Prepend "htm/" and sanitize the url
	url = "htm/" + url
	url = strings.ReplaceAll(url, "..", "X")
	log.Println("requested url:", url)
	data, err := os.ReadFile(url)
	if err != nil {
		log.Println("Error reading file:", url, err)
		return inf.defaultContents
	}
	return data
}