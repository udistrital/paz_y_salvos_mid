package services

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"time"
	_ "time/tzdata"

	beego "github.com/beego/beego/v2/server/web"
	"github.com/udistrital/paz_y_salvos_mid/models"
	"github.com/udistrital/utils_oas/v2/request"
)

var registroSNPGrado = regexp.MustCompile(`^[[:alnum:]-]+$`)

type ErrorInscripcionGrado struct {
	Status  int
	Mensaje string
}

func (e *ErrorInscripcionGrado) Error() string { return e.Mensaje }

func falloGrado(status int, mensaje string) error {
	return &ErrorInscripcionGrado{Status: status, Mensaje: mensaje}
}

type estudianteGrado struct {
	TerceroID int
	Codigo    string
	Ctx       context.Context
}

type programaGrado struct {
	Id               int         `json:"Id"`
	DependenciaId    int         `json:"DependenciaId"`
	Activo           bool        `json:"Activo"`
	NivelFormacionId *nivelGrado `json:"NivelFormacionId"`
}
type nivelGrado struct {
	Id                    int         `json:"Id"`
	NivelFormacionPadreId *nivelGrado `json:"NivelFormacionPadreId"`
}
type calendarioGrado struct {
	ProyectoId int `json:"ProyectoId"`
	Proceso    []struct {
		CodigoAbreviacion string        `json:"CodigoAbreviacion"`
		Eventos           []eventoGrado `json:"Eventos"`
	} `json:"Proceso"`
}
type eventoGrado struct {
	EventoId          int    `json:"EventoId"`
	CodigoAbreviacion string `json:"CodigoAbreviacion"`
	FechaInicioEvento string `json:"FechaInicioEvento"`
	FechaFinEvento    string `json:"FechaFinEvento"`
}

func baseGrado(clave string) (string, error) {
	v := strings.TrimSpace(beego.AppConfig.DefaultString(clave, ""))
	if v == "" {
		return "", falloGrado(http.StatusServiceUnavailable, "Servicio de inscripción no configurado: "+clave)
	}
	u, err := url.Parse(v)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil || u.RawQuery != "" || u.Fragment != "" {
		return "", falloGrado(http.StatusServiceUnavailable, "URL de inscripción inválida: "+clave)
	}
	return strings.TrimRight(v, "/") + "/", nil
}

func listaGrado[T any](raw json.RawMessage) ([]T, error) {
	raw = bytes.TrimSpace(raw)
	var envelope struct {
		Success bool            `json:"Success"`
		Data    json.RawMessage `json:"Data"`
		Status  json.RawMessage `json:"Status"`
	}
	if len(raw) > 0 && raw[0] == '[' {
		var items []T
		err := json.Unmarshal(raw, &items)
		return items, err
	}
	if err := json.Unmarshal(raw, &envelope); err != nil || !envelope.Success || !statusExternoValido(envelope.Status) {
		return nil, errors.New("respuesta externa inválida")
	}
	if len(envelope.Data) == 0 || bytes.Equal(bytes.TrimSpace(envelope.Data), []byte("null")) {
		return nil, errors.New("colección externa ausente")
	}
	var items []T
	if err := json.Unmarshal(envelope.Data, &items); err != nil {
		return nil, err
	}
	return items, nil
}

func statusExternoValido(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return true
	}
	valor := strings.Trim(string(raw), "\" ")
	n, err := strconv.Atoi(valor)
	return err == nil && n >= 200 && n < 300
}

func ctxAutorizado(ctx context.Context, token string) context.Context {
	return context.WithValue(ctx, "Authorization", token)
}

func resolverEstudiante(ctx context.Context, autorizacion string) (*estudianteGrado, error) {
	if !strings.HasPrefix(autorizacion, "Bearer ") || len(strings.TrimPrefix(autorizacion, "Bearer ")) < 10 {
		return nil, falloGrado(http.StatusUnauthorized, "Sesión no autenticada")
	}
	ctx = ctxAutorizado(ctx, autorizacion)
	infoURL, err := baseGrado("UrlAuthUserInfo")
	if err != nil {
		return nil, err
	}
	var info struct {
		Documento string          `json:"documento"`
		Email     string          `json:"email"`
		Codigo    string          `json:"Codigo"`
		Role      json.RawMessage `json:"role"`
	}
	status, err := request.GetWithContext(ctx, strings.TrimRight(infoURL, "/"), &info)
	if err != nil {
		if status == 400 || status == 401 || status == 403 {
			return nil, falloGrado(401, "La sesión no es válida. Inicia sesión nuevamente.")
		}
		return nil, falloGrado(503, "No fue posible contactar el servicio de autenticación. Intenta nuevamente.")
	}
	info.Email = strings.TrimSpace(info.Email)
	info.Documento = strings.TrimSpace(info.Documento)
	if info.Email == "" {
		return nil, falloGrado(http.StatusUnauthorized, "La sesión no contiene una identidad verificable")
	}
	var roles []string
	if err := json.Unmarshal(info.Role, &roles); err != nil {
		var texto string
		if json.Unmarshal(info.Role, &texto) == nil {
			roles = strings.Split(texto, ",")
		}
	}
	tieneRol := func(codigos []string) bool {
		for _, rol := range codigos {
			if strings.TrimSpace(rol) == "ESTUDIANTE" {
				return true
			}
		}
		return false
	}
	// El correo procede exclusivamente de userinfo validado con el token.
	if info.Codigo == "" || info.Documento == "" || !tieneRol(roles) {
		authURL, err := baseGrado("UrlAutenticacionMid")
		if err != nil {
			return nil, err
		}
		var payload json.RawMessage
		if _, err := request.PostWithContext(ctx, authURL+"token/userRol", map[string]string{"user": info.Email}, &payload); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo consultar el código estudiantil")
		}
		var perfil struct {
			Codigo    string   `json:"Codigo"`
			Documento string   `json:"documento"`
			Role      []string `json:"role"`
		}
		if err := json.Unmarshal(payload, &perfil); err != nil {
			return nil, falloGrado(http.StatusServiceUnavailable, "Respuesta de identidad inválida")
		}
		perfil.Documento = strings.TrimSpace(perfil.Documento)
		if info.Documento != "" && perfil.Documento != "" && perfil.Documento != info.Documento {
			return nil, falloGrado(http.StatusForbidden, "La identidad académica no coincide")
		}
		if info.Documento == "" {
			info.Documento = perfil.Documento
		}
		info.Codigo = perfil.Codigo
		roles = perfil.Role
	}
	if strings.TrimSpace(info.Codigo) == "" || info.Documento == "" || strings.ContainsAny(info.Documento, ",:|&") || !tieneRol(roles) {
		return nil, falloGrado(http.StatusForbidden, "No se acreditó la condición de estudiante")
	}
	tercerosURL, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return nil, err
	}
	query := url.Values{"query": {"Numero:" + info.Documento + ",Activo:true"}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, tercerosURL+"datos_identificacion?"+query.Encode(), &raw); err != nil {
		return nil, falloGrado(http.StatusServiceUnavailable, "No se pudo consultar Terceros")
	}
	var ident []struct {
		TerceroId struct {
			Id int `json:"Id"`
		} `json:"TerceroId"`
	}
	ident, err = listaGrado[struct {
		TerceroId struct {
			Id int `json:"Id"`
		} `json:"TerceroId"`
	}](raw)
	if err != nil || len(ident) == 0 {
		return nil, falloGrado(http.StatusForbidden, "Tercero del estudiante no verificable")
	}
	terceroID := ident[0].TerceroId.Id
	for _, dato := range ident {
		if terceroID <= 0 || dato.TerceroId.Id != terceroID {
			return nil, falloGrado(403, "Tercero del estudiante no verificable")
		}
	}
	return &estudianteGrado{TerceroID: terceroID, Codigo: strings.TrimSpace(info.Codigo), Ctx: ctx}, nil
}

func resolverEstadoBorrador(ctx context.Context) (int, error) {
	return resolverParametroGrado(ctx, "EST_SOL_GRADO", "SG_BORRADOR")
}

func resolverEstadoRadicada(ctx context.Context) (int, error) {
	return resolverParametroGrado(ctx, "EST_SOL_GRADO", "SG_RADICADA")
}

func resolverParametroGrado(ctx context.Context, tipoCodigo, codigo string) (int, error) {
	paramURL, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return 0, err
	}
	query := url.Values{"query": {"CodigoAbreviacion:" + codigo + ",Activo:true,TipoParametroId.CodigoAbreviacion:" + tipoCodigo + ",TipoParametroId.Activo:true,TipoParametroId.AreaTipoId.CodigoAbreviacion:PSGA,TipoParametroId.AreaTipoId.Activo:true"}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, paramURL+"parametro?"+query.Encode(), &raw); err != nil {
		return 0, falloGrado(http.StatusServiceUnavailable, "No se pudo resolver "+codigo)
	}
	parametros, err := listaGrado[struct {
		Id              int `json:"Id"`
		TipoParametroId struct {
			Id                int    `json:"Id"`
			CodigoAbreviacion string `json:"CodigoAbreviacion"`
		} `json:"TipoParametroId"`
	}](raw)
	if err != nil || len(parametros) != 1 || parametros[0].Id <= 0 {
		return 0, falloGrado(http.StatusServiceUnavailable, "Parámetro "+codigo+" ausente o duplicado")
	}
	var tipo struct {
		Success bool `json:"Success"`
		Data    struct {
			CodigoAbreviacion string `json:"CodigoAbreviacion"`
			Activo            bool   `json:"Activo"`
			AreaTipoId        struct {
				Id int `json:"Id"`
			} `json:"AreaTipoId"`
		} `json:"Data"`
	}
	if _, err := request.GetWithContext(ctx, paramURL+"tipo_parametro/"+strconv.Itoa(parametros[0].TipoParametroId.Id), &tipo); err != nil || !tipo.Success || !tipo.Data.Activo || tipo.Data.CodigoAbreviacion != tipoCodigo {
		return 0, falloGrado(http.StatusServiceUnavailable, "Tipo de "+codigo+" inválido")
	}
	var area struct {
		Success bool `json:"Success"`
		Data    struct {
			CodigoAbreviacion string `json:"CodigoAbreviacion"`
			Activo            bool   `json:"Activo"`
		} `json:"Data"`
	}
	if _, err := request.GetWithContext(ctx, paramURL+"area_tipo/"+strconv.Itoa(tipo.Data.AreaTipoId.Id), &area); err != nil || !area.Success || !area.Data.Activo || area.Data.CodigoAbreviacion != "PSGA" {
		return 0, falloGrado(http.StatusServiceUnavailable, codigo+" no pertenece a PSGA")
	}
	return parametros[0].Id, nil
}

func fechaGradoServidor(s string) (time.Time, error) {
	if s == "" || strings.HasPrefix(s, "0001-") {
		return time.Time{}, errors.New("fecha ausente")
	}
	if t, err := time.Parse(time.RFC3339Nano, s); err == nil {
		return t, nil
	}
	bogota, err := time.LoadLocation("America/Bogota")
	if err != nil {
		return time.Time{}, err
	}
	for _, formato := range []string{"2006-01-02T15:04:05.999999999", "2006-01-02 15:04:05.999999999", "2006-01-02"} {
		if t, err := time.ParseInLocation(formato, s, bogota); err == nil {
			return t, nil
		}
	}
	return time.Time{}, errors.New("fecha inválida")
}

func vigenteGrado(desde, hasta string, ahora time.Time) bool {
	inicio, err := fechaGradoServidor(desde)
	if err != nil {
		return false
	}
	fin, err := fechaGradoServidor(hasta)
	return err == nil && !fin.Before(inicio) && !ahora.Before(inicio) && !ahora.After(fin)
}

func resolverProgramaGrado(ctx context.Context, terceroID, programaID int) (*programaGrado, error) {
	if programaID <= 0 {
		return nil, falloGrado(http.StatusBadRequest, "Programa obligatorio")
	}
	paramURL, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return nil, err
	}
	var paramRaw json.RawMessage
	query := url.Values{"query": {"CodigoAbreviacion:EST,TipoParametroId.CodigoAbreviacion:TV,Activo:true"}, "limit": {"100"}}
	if _, err := request.GetWithContext(ctx, paramURL+"parametro?"+query.Encode(), &paramRaw); err != nil {
		return nil, falloGrado(503, "Vinculación estudiantil no verificable")
	}
	tipos, err := listaGrado[struct {
		Id int `json:"Id"`
	}](paramRaw)
	if err != nil || len(tipos) != 1 || tipos[0].Id <= 0 {
		return nil, falloGrado(503, "Tipo de vinculación estudiantil ambiguo")
	}
	tercerosURL, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return nil, err
	}
	q := url.Values{"query": {fmt.Sprintf("TerceroPrincipalId.Id:%d,TipoVinculacionId:%d,Activo:true", terceroID, tipos[0].Id)}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, tercerosURL+"vinculacion?"+q.Encode(), &raw); err != nil {
		return nil, falloGrado(503, "No se pudo verificar la vinculación")
	}
	vinculos, err := listaGrado[struct {
		DependenciaId          int    `json:"DependenciaId"`
		Activo                 bool   `json:"Activo"`
		FechaInicioVinculacion string `json:"FechaInicioVinculacion"`
		FechaFinVinculacion    string `json:"FechaFinVinculacion"`
	}](raw)
	if err != nil {
		return nil, falloGrado(503, "Vinculación inválida")
	}
	proyectosURL, err := baseGrado("UrlProyectoAcademico")
	if err != nil {
		return nil, err
	}
	var encontrado *programaGrado
	for _, v := range vinculos {
		if !v.Activo || v.DependenciaId <= 0 {
			continue
		}
		if v.FechaInicioVinculacion != "" && !strings.HasPrefix(v.FechaInicioVinculacion, "0001-") {
			d, e := fechaGradoServidor(v.FechaInicioVinculacion)
			if e != nil || time.Now().Before(d) {
				continue
			}
		}
		if v.FechaFinVinculacion != "" && !strings.HasPrefix(v.FechaFinVinculacion, "0001-") {
			d, e := fechaGradoServidor(v.FechaFinVinculacion)
			if e != nil || time.Now().After(d) {
				continue
			}
		}
		q := url.Values{"query": {fmt.Sprintf("DependenciaId:%d,Activo:true", v.DependenciaId)}, "limit": {"100"}}
		var proyectosRaw json.RawMessage
		if _, err := request.GetWithContext(ctx, proyectosURL+"proyecto_academico_institucion?"+q.Encode(), &proyectosRaw); err != nil {
			return nil, falloGrado(503, "No se pudo validar el programa")
		}
		proyectos, e := listaGrado[programaGrado](proyectosRaw)
		if e != nil {
			return nil, falloGrado(503, "Respuesta de programa inválida")
		}
		for _, p := range proyectos {
			if p.Id == programaID && p.DependenciaId == v.DependenciaId && p.Activo {
				if encontrado != nil && encontrado.DependenciaId != p.DependenciaId {
					return nil, falloGrado(409, "Programa ambiguo")
				}
				copia := p
				encontrado = &copia
			}
		}
	}
	if encontrado == nil {
		return nil, falloGrado(http.StatusForbidden, "El programa no pertenece al estudiante")
	}
	return encontrado, nil
}

// resolverCodigoPrograma cruza los códigos CODE del tercero con la carrera de
// Académica y su dependencia homologada. El código único del login no identifica
// necesariamente el programa cuando un estudiante cursa más de uno.
func resolverCodigoPrograma(ctx context.Context, terceroID, dependenciaOikosID int) (string, error) {
	if terceroID <= 0 || dependenciaOikosID <= 0 {
		return "", falloGrado(400, "Estudiante o dependencia inválidos")
	}
	tercerosURL, err := baseGrado("UrlTercerosCrud")
	if err != nil {
		return "", err
	}
	q := url.Values{"query": {fmt.Sprintf("Activo:true,TerceroId.Id:%d,TipoDocumentoId.CodigoAbreviacion:CODE", terceroID)}, "limit": {"100"}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, tercerosURL+"datos_identificacion?"+q.Encode(), &raw); err != nil {
		return "", falloGrado(503, "No se pudieron consultar los códigos estudiantiles")
	}
	identificaciones, err := listaGrado[struct {
		Numero    string `json:"Numero"`
		TerceroId struct {
			Id int `json:"Id"`
		} `json:"TerceroId"`
	}](raw)
	if err != nil || len(identificaciones) == 0 || len(identificaciones) >= 100 {
		return "", falloGrado(503, "Códigos estudiantiles ausentes o incompletos")
	}
	wsURL, err := baseGrado("UrlcrudWSO2")
	if err != nil {
		return "", err
	}
	academica := strings.Trim(beego.AppConfig.DefaultString("NscrudAcademica", ""), "/")
	homologacion := strings.Trim(beego.AppConfig.DefaultString("NscrudHomologacion", ""), "/")
	if academica == "" || homologacion == "" {
		return "", falloGrado(503, "Fuentes académicas no configuradas")
	}
	vistos := make(map[string]bool)
	var correspondiente string
	for _, identificacion := range identificaciones {
		codigo := strings.TrimSpace(identificacion.Numero)
		if codigo == "" || identificacion.TerceroId.Id != terceroID || strings.ContainsAny(codigo, "/?#") {
			return "", falloGrado(503, "Códigos estudiantiles inconsistentes")
		}
		if vistos[codigo] {
			continue
		} // La fuente puede repetir identificaciones CODE.
		vistos[codigo] = true
		var estudiante struct {
			EstudianteCollection struct {
				DatosEstudiante []struct {
					Codigo  string `json:"codigo"`
					Carrera string `json:"carrera"`
				} `json:"datosEstudiante"`
			} `json:"estudianteCollection"`
		}
		status, err := request.GetWithContext(ctx, wsURL+academica+"/datos_estudiante/"+url.PathEscape(codigo), &estudiante)
		if status == 404 {
			continue
		} // Identificación CODE histórica sin registro académico.
		if err != nil {
			return "", falloGrado(503, "No se pudo verificar la carrera de un código estudiantil")
		}
		for _, dato := range estudiante.EstudianteCollection.DatosEstudiante {
			if strings.TrimSpace(dato.Codigo) != codigo || dato.Carrera == "" || strings.ContainsAny(dato.Carrera, "/?#") {
				return "", falloGrado(503, "Respuesta académica inconsistente")
			}
			var respuesta models.HomologacionResponse
			status, err := request.GetWithContext(ctx, wsURL+homologacion+"/proyecto_curricular_cod_proyecto/"+url.PathEscape(dato.Carrera), &respuesta)
			if status == 404 {
				continue
			}
			if err != nil || respuesta.OikosId() <= 0 {
				return "", falloGrado(503, "No se pudo homologar la carrera del código")
			}
			if respuesta.OikosId() == dependenciaOikosID {
				if correspondiente != "" && correspondiente != codigo {
					return "", falloGrado(409, "Hay varios códigos del estudiante para el programa")
				}
				correspondiente = codigo
			}
		}
	}
	if correspondiente == "" {
		return "", falloGrado(403, "No se encontró un código estudiantil para el programa seleccionado")
	}
	return correspondiente, nil
}

func eventosGrado(ctx context.Context, programa *programaGrado, periodoID int) (eventoGrado, eventoGrado, error) {
	var cero eventoGrado
	if periodoID <= 0 || programa.NivelFormacionId == nil {
		return cero, cero, falloGrado(400, "Periodo o nivel inválido")
	}
	nivel := programa.NivelFormacionId
	visitados := make(map[int]bool)
	for nivel.NivelFormacionPadreId != nil {
		if visitados[nivel.Id] {
			return cero, cero, falloGrado(503, "Jerarquía de formación inválida")
		}
		visitados[nivel.Id] = true
		nivel = nivel.NivelFormacionPadreId
	}
	if nivel.Id <= 0 {
		return cero, cero, falloGrado(503, "Nivel de formación inválido")
	}
	calURL, err := baseGrado("UrlCalendarioMid")
	if err != nil {
		return cero, cero, err
	}
	query := url.Values{"id-nivel": {strconv.Itoa(nivel.Id)}, "id-periodo": {strconv.Itoa(periodoID)}}
	var raw json.RawMessage
	if _, err := request.GetWithContext(ctx, calURL+"calendario-proyecto/calendario/proyecto?"+query.Encode(), &raw); err != nil {
		return cero, cero, falloGrado(503, "Calendario no verificable")
	}
	calendarios, err := listaGrado[calendarioGrado](raw)
	if err != nil {
		return cero, cero, falloGrado(503, "Respuesta de Calendario inválida")
	}
	var ins, apr []eventoGrado
	procesos := 0
	coincidenciasPrograma := 0
	for _, c := range calendarios {
		if c.ProyectoId != programa.Id {
			continue
		}
		coincidenciasPrograma++
		for _, p := range c.Proceso {
			if p.CodigoAbreviacion != "PROC_GRAD" {
				continue
			}
			procesos++
			for _, ev := range p.Eventos {
				switch ev.CodigoAbreviacion {
				case "INSC_GRADO":
					ins = append(ins, ev)
				case "APROB_PAZ_SALVO":
					apr = append(apr, ev)
				}
			}
		}
	}
	if coincidenciasPrograma != 1 || procesos != 1 || len(ins) != 1 || len(apr) != 1 || ins[0].EventoId <= 0 || apr[0].EventoId <= 0 {
		return cero, cero, falloGrado(503, "Eventos de Grado ausentes o ambiguos")
	}
	for _, ev := range []eventoGrado{ins[0], apr[0]} {
		d, e := fechaGradoServidor(ev.FechaInicioEvento)
		f, e2 := fechaGradoServidor(ev.FechaFinEvento)
		if e != nil || e2 != nil || f.Before(d) {
			return cero, cero, falloGrado(503, "Fechas de Calendario inválidas")
		}
	}
	return ins[0], apr[0], nil
}

func validarVentana(ins, apr eventoGrado, primera bool) error {
	now := time.Now()
	if !vigenteGrado(apr.FechaInicioEvento, apr.FechaFinEvento, now) {
		if inicio, err := fechaGradoServidor(apr.FechaInicioEvento); err == nil && now.Before(inicio) {
			return falloGrado(409, "Los tiempos de aprobación aún no han iniciado")
		}
		return falloGrado(409, "Los tiempos de aprobación se han cerrado")
	}
	if primera && !vigenteGrado(ins.FechaInicioEvento, ins.FechaFinEvento, now) {
		return falloGrado(409, "La inscripción a grado no está abierta")
	}
	return nil
}

func contenidoGradoValido(raw json.RawMessage) bool {
	if !json.Valid(raw) {
		return false
	}
	var obj map[string]json.RawMessage
	return json.Unmarshal(raw, &obj) == nil && obj != nil && !bytes.Equal(bytes.TrimSpace(raw), []byte("null"))
}

func CrearBorradorGrado(ctx context.Context, auth string, entrada models.CrearBorradorGrado) (*models.BorradorGrado, error) {
	if entrada.PeriodoId <= 0 || entrada.ProgramaAcademicoId <= 0 || !contenidoGradoValido(entrada.Contenido) {
		return nil, falloGrado(400, "Datos del borrador inválidos")
	}
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return nil, err
	}
	if err := validarPeriodoGrado(user.Ctx, entrada.PeriodoId); err != nil {
		return nil, err
	}
	programa, err := resolverProgramaGrado(user.Ctx, user.TerceroID, entrada.ProgramaAcademicoId)
	if err != nil {
		return nil, err
	}
	codigo, err := resolverCodigoPrograma(user.Ctx, user.TerceroID, programa.DependenciaId)
	if err != nil {
		return nil, err
	}
	ins, apr, err := eventosGrado(user.Ctx, programa, entrada.PeriodoId)
	if err != nil {
		return nil, err
	}
	if err := validarVentana(ins, apr, true); err != nil {
		return nil, err
	}
	estado, err := resolverEstadoBorrador(user.Ctx)
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	body := map[string]interface{}{"TerceroId": user.TerceroID, "CodigoEstudiante": codigo, "PeriodoId": entrada.PeriodoId, "ProgramaAcademicoId": programa.Id, "DependenciaOikosId": programa.DependenciaId, "CalendarioEventoInscripcionId": ins.EventoId, "CalendarioEventoAprobacionId": apr.EventoId, "EstadoBorradorId": estado, "Contenido": entrada.Contenido}
	var resp models.APIResponseData[models.BorradorGrado]
	status, err := request.PostWithContext(user.Ctx, base+"solicitud-grado/borrador", body, &resp)
	if err != nil {
		if status == 409 {
			return nil, falloGrado(409, "Ya existe una solicitud activa en este periodo y programa")
		}
		return nil, falloGrado(503, "No se pudo guardar el borrador")
	}
	if !resp.Success || resp.Status >= 400 || resp.Data.Solicitud.Id <= 0 || resp.Data.Solicitud.TerceroId != user.TerceroID || resp.Data.Solicitud.CodigoEstudiante != codigo || resp.Data.Solicitud.PeriodoId != entrada.PeriodoId || resp.Data.Solicitud.ProgramaAcademicoId != entrada.ProgramaAcademicoId {
		return nil, falloGrado(503, "Respuesta de borrador inválida")
	}
	return &resp.Data, nil
}

func ObtenerBorradorGrado(ctx context.Context, auth string, periodoID, programaID int) (*models.BorradorGrado, error) {
	if periodoID <= 0 || programaID <= 0 {
		return nil, falloGrado(400, "Periodo y programa requeridos")
	}
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return nil, err
	}
	// La lectura se limita al titular en CRUD. Una convocatoria cerrada no oculta
	// una solicitud que ya pertenece al estudiante, incluso después de radicarla.
	estadoBorrador, err := resolverEstadoBorrador(user.Ctx)
	if err != nil {
		return nil, err
	}
	estadoRadicada, err := resolverEstadoRadicada(user.Ctx)
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	q := url.Values{"tercero_id": {strconv.Itoa(user.TerceroID)}, "periodo_id": {strconv.Itoa(periodoID)}, "programa_id": {strconv.Itoa(programaID)},
		"estado_borrador_id": {strconv.Itoa(estadoBorrador)}, "estado_radicada_id": {strconv.Itoa(estadoRadicada)}}
	var resp models.APIResponseData[models.BorradorGrado]
	status, err := request.GetWithContext(user.Ctx, base+"solicitud-grado/borrador?"+q.Encode(), &resp)
	if err != nil {
		if status == 404 {
			return nil, falloGrado(404, "Borrador no encontrado")
		}
		if status == 409 {
			return nil, falloGrado(409, "La solicitud no está disponible para consulta")
		}
		return nil, falloGrado(503, "No se pudo consultar la solicitud")
	}
	if !resp.Success || resp.Status >= 400 || resp.Data.Solicitud.Id <= 0 || resp.Data.Solicitud.TerceroId != user.TerceroID || resp.Data.Solicitud.PeriodoId != periodoID || resp.Data.Solicitud.ProgramaAcademicoId != programaID {
		return nil, falloGrado(503, "Respuesta de solicitud inválida")
	}
	codigo, err := resolverCodigoPrograma(user.Ctx, user.TerceroID, resp.Data.Solicitud.DependenciaOikosId)
	if err != nil {
		return nil, err
	}
	if resp.Data.Solicitud.CodigoEstudiante != codigo {
		return nil, falloGrado(409, "La solicitud tiene un código estudiantil de otro programa; requiere corrección")
	}
	return &resp.Data, nil
}

func GuardarBorradorGrado(ctx context.Context, auth string, id int, contenido json.RawMessage) (*models.BorradorGrado, error) {
	if id <= 0 || !contenidoGradoValido(contenido) {
		return nil, falloGrado(400, "Contenido inválido")
	}
	user, err := resolverEstudiante(ctx, auth)
	if err != nil {
		return nil, err
	}
	estado, err := resolverEstadoBorrador(user.Ctx)
	if err != nil {
		return nil, err
	}
	base, err := baseGrado("UrlCrudPazySalvos")
	if err != nil {
		return nil, err
	}
	q := url.Values{"tercero_id": {strconv.Itoa(user.TerceroID)}, "estado_borrador_id": {strconv.Itoa(estado)}}
	path := base + "solicitud-grado/borrador/" + strconv.Itoa(id) + "?" + q.Encode()
	var resp models.APIResponseData[models.BorradorGrado]
	status, err := request.GetWithContext(user.Ctx, path, &resp)
	if err != nil {
		if status == 404 {
			return nil, falloGrado(404, "Borrador no encontrado")
		}
		if status == 409 {
			return nil, falloGrado(409, "Borrador no editable")
		}
		return nil, falloGrado(503, "No se pudo consultar el borrador")
	}
	if !resp.Success || resp.Status >= 400 || resp.Data.Solicitud.Id != id || resp.Data.Solicitud.TerceroId != user.TerceroID {
		return nil, falloGrado(403, "Borrador ajeno")
	}
	programa, err := resolverProgramaGrado(user.Ctx, user.TerceroID, resp.Data.Solicitud.ProgramaAcademicoId)
	if err != nil {
		return nil, err
	}
	if resp.Data.Solicitud.DependenciaOikosId != programa.DependenciaId {
		return nil, falloGrado(409, "La dependencia del borrador no coincide con el programa")
	}
	codigo, err := resolverCodigoPrograma(user.Ctx, user.TerceroID, programa.DependenciaId)
	if err != nil {
		return nil, err
	}
	if resp.Data.Solicitud.CodigoEstudiante != codigo {
		return nil, falloGrado(409, "El borrador tiene un código estudiantil de otro programa; requiere corrección")
	}
	_, apr, err := eventosGrado(user.Ctx, programa, resp.Data.Solicitud.PeriodoId)
	if err != nil {
		return nil, err
	}
	if err := validarVentana(eventoGrado{}, apr, false); err != nil {
		return nil, err
	}
	status, err = request.PutWithContext(user.Ctx, path, models.GuardarBorradorGrado{Contenido: contenido}, &resp)
	if err != nil {
		if status == 409 {
			return nil, falloGrado(409, "La versión ya no es editable")
		}
		return nil, falloGrado(503, "No se pudo actualizar el borrador")
	}
	if !resp.Success || resp.Status >= 400 || resp.Data.Solicitud.Id != id || resp.Data.Solicitud.TerceroId != user.TerceroID {
		return nil, falloGrado(503, "Respuesta de actualización inválida")
	}
	return &resp.Data, nil
}

func validarPeriodoGrado(ctx context.Context, id int) error {
	base, err := baseGrado("UrlcrudParametros")
	if err != nil {
		return err
	}
	var respuesta struct {
		Success bool
		Data    struct {
			Id                int
			Activo            bool
			CodigoAbreviacion string
		}
	}
	status, err := request.GetWithContext(ctx, base+"periodo/"+strconv.Itoa(id), &respuesta)
	if status == 404 {
		return falloGrado(400, "Periodo académico inexistente")
	}
	if err != nil || !respuesta.Success {
		return falloGrado(503, "No se pudo validar el periodo académico")
	}
	if respuesta.Data.Id != id || !respuesta.Data.Activo || respuesta.Data.CodigoAbreviacion != "PA" {
		return falloGrado(400, "Periodo académico no habilitado")
	}
	return nil
}
