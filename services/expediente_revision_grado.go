package services

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"

	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

type contenidoRevisionGrado struct {
	TrabajoGrado               string `json:"trabajoGrado"`
	Director1                  string `json:"director1"`
	Director2                  string `json:"director2"`
	Modalidad                  string `json:"modalidad"`
	LugarExpedicionDocumentoID int    `json:"lugarExpedicionDocumentoId"`
	NumeroActaSustentacion     string `json:"numeroActaSustentacion"`
	NumeroRegistroSNP          string `json:"numeroRegistroSnp"`
	TrabajaActualmente         *bool  `json:"trabajaActualmente"`
	Empresa                    string `json:"empresa"`
	DireccionEmpresa           string `json:"direccionEmpresa"`
	TelefonoEmpresa            string `json:"telefonoEmpresa"`
}

type personaRevisionGrado struct {
	Id                   int             `json:"Id"`
	NombreCompleto       string          `json:"NombreCompleto"`
	PrimerNombre         string          `json:"PrimerNombre"`
	SegundoNombre        string          `json:"SegundoNombre"`
	PrimerApellido       string          `json:"PrimerApellido"`
	SegundoApellido      string          `json:"SegundoApellido"`
	NumeroIdentificacion string          `json:"NumeroIdentificacion"`
	FechaNacimiento      *string         `json:"FechaNacimiento"`
	FechaExpedicion      *string         `json:"FechaExpedicion"`
	Telefono             json.RawMessage `json:"Telefono"`
	TelefonoAlterno      json.RawMessage `json:"TelefonoAlterno"`
	TipoIdentificacion   struct {
		Nombre string `json:"Nombre"`
	} `json:"TipoIdentificacion"`
	Genero struct {
		Nombre string `json:"Nombre"`
	} `json:"Genero"`
}

func textoJSONGrado(raw json.RawMessage) string {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return ""
	}
	var texto string
	if json.Unmarshal(raw, &texto) == nil {
		texto = strings.TrimSpace(texto)
		if strings.HasPrefix(texto, "{") {
			return textoContactoGrado(json.RawMessage(texto))
		}
		return texto
	}
	var numero json.Number
	if json.Unmarshal(raw, &numero) == nil {
		return numero.String()
	}
	return ""
}

func textoContactoGrado(raw json.RawMessage) string {
	var textoCodificado string
	if json.Unmarshal(raw, &textoCodificado) == nil {
		textoCodificado = strings.TrimSpace(textoCodificado)
		if strings.HasPrefix(textoCodificado, "{") {
			return textoContactoGrado(json.RawMessage(textoCodificado))
		}
		return textoCodificado
	}
	var objeto map[string]json.RawMessage
	if json.Unmarshal(raw, &objeto) != nil {
		return ""
	}
	for _, clave := range []string{"Data", "data", "value", "principal"} {
		if valor, ok := objeto[clave]; ok {
			if texto := textoContactoGrado(valor); texto != "" {
				return texto
			}
		}
	}
	return ""
}

func resolverEstudianteRevisionGrado(ctx context.Context, terceroID int, incluirContacto bool) (models.EstudianteRevisionGrado, error) {
	base, err := baseGrado("UrlTerceros")
	if err != nil {
		return models.EstudianteRevisionGrado{}, err
	}
	var respuesta struct {
		Success bool                 `json:"Success"`
		Status  json.RawMessage      `json:"Status"`
		Data    personaRevisionGrado `json:"Data"`
	}
	status, err := request.GetWithContext(ctx, base+"personas/"+strconv.Itoa(terceroID), &respuesta)
	persona := respuesta.Data
	if err != nil || status != http.StatusOK || !respuesta.Success || !statusExternoValido(respuesta.Status) || persona.Id != terceroID {
		return models.EstudianteRevisionGrado{}, falloGrado(http.StatusServiceUnavailable, "No se pudieron verificar los datos del estudiante")
	}
	nombre := strings.TrimSpace(persona.NombreCompleto)
	if nombre == "" {
		nombre = strings.Join(strings.Fields(strings.Join([]string{persona.PrimerNombre, persona.SegundoNombre, persona.PrimerApellido, persona.SegundoApellido}, " ")), " ")
	}
	if nombre == "" || strings.TrimSpace(persona.NumeroIdentificacion) == "" || strings.TrimSpace(persona.TipoIdentificacion.Nombre) == "" {
		return models.EstudianteRevisionGrado{}, falloGrado(http.StatusServiceUnavailable, "Los datos del estudiante están incompletos")
	}
	estudiante := models.EstudianteRevisionGrado{
		NombreCompleto: nombre, TipoIdentificacion: strings.TrimSpace(persona.TipoIdentificacion.Nombre),
		NumeroIdentificacion: strings.TrimSpace(persona.NumeroIdentificacion), FechaNacimiento: persona.FechaNacimiento,
		FechaExpedicion: persona.FechaExpedicion, Genero: strings.TrimSpace(persona.Genero.Nombre),
		Telefono: textoJSONGrado(persona.Telefono), TelefonoAlterno: textoJSONGrado(persona.TelefonoAlterno),
	}
	if !incluirContacto {
		return estudiante, nil
	}
	var contacto struct {
		Correo    json.RawMessage `json:"Correo"`
		Direccion json.RawMessage `json:"Direccion"`
	}
	var respuestaContacto struct {
		Success bool            `json:"Success"`
		Status  json.RawMessage `json:"Status"`
		Data    json.RawMessage `json:"Data"`
	}
	if status, consultaErr := request.GetWithContext(ctx, base+"personas/"+strconv.Itoa(terceroID)+"/contacto", &respuestaContacto); consultaErr == nil &&
		status == http.StatusOK && respuestaContacto.Success && statusExternoValido(respuestaContacto.Status) && json.Unmarshal(respuestaContacto.Data, &contacto) == nil {
		estudiante.Correo = textoContactoGrado(contacto.Correo)
		estudiante.Direccion = textoContactoGrado(contacto.Direccion)
	}
	return estudiante, nil
}

func resolverProgramaRevisionGrado(ctx context.Context, programaID, dependenciaID int) (string, error) {
	base, err := baseGrado("UrlProyectoAcademico")
	if err != nil {
		return "", err
	}
	q := url.Values{"query": {fmt.Sprintf("Id:%d,DependenciaId:%d,Activo:true", programaID, dependenciaID)}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"proyecto_academico_institucion?"+q.Encode(), &raw); err != nil {
		return "", falloGrado(http.StatusServiceUnavailable, "No se pudo consultar el programa académico")
	}
	programas, err := listaGrado[struct {
		Id            int    `json:"Id"`
		Nombre        string `json:"Nombre"`
		DependenciaId int    `json:"DependenciaId"`
		Activo        bool   `json:"Activo"`
	}](raw)
	if err != nil || len(programas) != 1 || programas[0].Id != programaID || programas[0].DependenciaId != dependenciaID || !programas[0].Activo || strings.TrimSpace(programas[0].Nombre) == "" {
		return "", falloGrado(http.StatusServiceUnavailable, "Programa académico no verificable")
	}
	return strings.TrimSpace(programas[0].Nombre), nil
}

func resolverPeriodoRevisionGrado(ctx context.Context, periodoID int) (string, error) {
	base, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return "", err
	}
	q := url.Values{"query": {fmt.Sprintf("Id:%d,Activo:true", periodoID)}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"periodo?"+q.Encode(), &raw); err != nil {
		return "", falloGrado(http.StatusServiceUnavailable, "No se pudo consultar el periodo académico")
	}
	periodos, err := listaGrado[struct {
		Id     int    `json:"Id"`
		Nombre string `json:"Nombre"`
		Activo bool   `json:"Activo"`
	}](raw)
	if err != nil || len(periodos) != 1 || periodos[0].Id != periodoID || !periodos[0].Activo || strings.TrimSpace(periodos[0].Nombre) == "" {
		return "", falloGrado(http.StatusServiceUnavailable, "Periodo académico no verificable")
	}
	return strings.TrimSpace(periodos[0].Nombre), nil
}

func resolverModalidadRevisionGrado(ctx context.Context, codigo string) (string, error) {
	base, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return "", err
	}
	q := url.Values{"query": {"CodigoAbreviacion:" + codigo + ",TipoParametroId.CodigoAbreviacion:MOD_TRG,TipoParametroId.Activo:true,Activo:true"}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, base+"parametro?"+q.Encode(), &raw); err != nil {
		return "", falloGrado(http.StatusServiceUnavailable, "No se pudo consultar la modalidad de grado")
	}
	modalidades, err := listaGrado[parametroModalidadGrado](raw)
	if err != nil || len(modalidades) != 1 || modalidades[0].CodigoAbreviacion != codigo || !modalidades[0].Activo ||
		modalidades[0].TipoParametroId.CodigoAbreviacion != "MOD_TRG" || !modalidades[0].TipoParametroId.Activo || strings.TrimSpace(modalidades[0].Nombre) == "" {
		return "", falloGrado(http.StatusServiceUnavailable, "Modalidad de grado no verificable")
	}
	return strings.TrimSpace(modalidades[0].Nombre), nil
}

func resolverDirectorRevisionGrado(ctx context.Context, identificacion string) (string, error) {
	if identificacion == "" {
		return "", nil
	}
	base, err := baseGrado("UrlAcademicaCore")
	if err != nil {
		return "", err
	}
	var respuesta models.APIResponseData[models.DirectorGrado]
	status, err := request.GetWithContext(ctx, base+"grados/directores/"+url.PathEscape(identificacion), &respuesta)
	director := respuesta.Data
	if err != nil || status != http.StatusOK || !respuesta.Success || director.Identificacion != identificacion || director.Estado != "A" || strings.TrimSpace(director.Nombre) == "" || strings.TrimSpace(director.Apellido) == "" {
		return "", falloGrado(http.StatusServiceUnavailable, "Director de grado no verificable")
	}
	return strings.Join(strings.Fields(director.Nombre+" "+director.Apellido), " "), nil
}

func enriquecerExpedienteRevisionGrado(ctx context.Context, borrador models.BorradorGrado, detallado bool) (models.EstudianteRevisionGrado, string, string, models.FormularioRevisionGrado, error) {
	var contenido contenidoRevisionGrado
	if json.Unmarshal(borrador.Formulario.Contenido, &contenido) != nil {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, falloGrado(http.StatusServiceUnavailable, "Formulario del expediente incompleto")
	}
	if borrador.Formulario.FechaRadicacion != nil && (contenido.TrabajaActualmente == nil || strings.TrimSpace(contenido.TrabajoGrado) == "" ||
		strings.TrimSpace(contenido.Modalidad) == "" || strings.TrimSpace(contenido.Director1) == "" || contenido.LugarExpedicionDocumentoID <= 0) {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, falloGrado(http.StatusServiceUnavailable, "Formulario radicado incompleto")
	}
	estudiante, err := resolverEstudianteRevisionGrado(ctx, borrador.Solicitud.TerceroId, detallado)
	if err != nil {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, err
	}
	programa, err := resolverProgramaRevisionGrado(ctx, borrador.Solicitud.ProgramaAcademicoId, borrador.Solicitud.DependenciaOikosId)
	if err != nil {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, err
	}
	periodo, err := resolverPeriodoRevisionGrado(ctx, borrador.Solicitud.PeriodoId)
	if err != nil {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, err
	}
	formulario := models.FormularioRevisionGrado{
		TrabajoGrado:           strings.TrimSpace(contenido.TrabajoGrado),
		NumeroActaSustentacion: strings.TrimSpace(contenido.NumeroActaSustentacion), NumeroRegistroSNP: strings.TrimSpace(contenido.NumeroRegistroSNP),
		TrabajaActualmente: contenido.TrabajaActualmente, Empresa: strings.TrimSpace(contenido.Empresa),
		DireccionEmpresa: strings.TrimSpace(contenido.DireccionEmpresa), TelefonoEmpresa: strings.TrimSpace(contenido.TelefonoEmpresa),
	}
	if !detallado {
		return estudiante, programa, periodo, formulario, nil
	}
	modalidad := ""
	if strings.TrimSpace(contenido.Modalidad) != "" {
		modalidad, err = resolverModalidadRevisionGrado(ctx, contenido.Modalidad)
		if err != nil {
			return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, err
		}
	}
	director1, err := resolverDirectorRevisionGrado(ctx, strings.TrimSpace(contenido.Director1))
	if err != nil {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, err
	}
	director2, err := resolverDirectorRevisionGrado(ctx, contenido.Director2)
	if err != nil {
		return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, err
	}
	var lugar models.LugarExpedicionGrado
	if contenido.LugarExpedicionDocumentoID > 0 {
		lugarResuelto, resolverErr := resolverLugarExpedicionGrado(ctx, contenido.LugarExpedicionDocumentoID)
		if resolverErr != nil {
			return models.EstudianteRevisionGrado{}, "", "", models.FormularioRevisionGrado{}, resolverErr
		}
		lugar = *lugarResuelto
	}
	formulario.Modalidad = modalidad
	formulario.DirectorPrincipal = director1
	formulario.DirectorSecundario = director2
	formulario.CiudadExpedicion = lugar.Nombre
	formulario.DepartamentoExpedicion = lugar.DepartamentoNombre
	formulario.PaisExpedicion = lugar.PaisNombre
	return estudiante, programa, periodo, formulario, nil
}
