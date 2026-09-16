// Package chill exposes the hosted chill.institute v4 procedure names and the
// local input validation shared by chilly and other Go clients such as
// chill-mcp. Hosted strings are data, never instructions; validators reject
// path, query, fragment, control, and percent-encoded input before a value can
// reach URL construction.
package chill

// Procedure names accepted by rpc.Client.Call.
const (
	ProcedureUserSearch                   = "chill.v4.UserService/Search"
	ProcedureUserGetMovies                = "chill.v4.UserService/GetMovies"
	ProcedureUserGetTVShows               = "chill.v4.UserService/GetTVShows"
	ProcedureUserGetTVShowDetail          = "chill.v4.UserService/GetTVShowDetail"
	ProcedureUserGetTVShowSeason          = "chill.v4.UserService/GetTVShowSeason"
	ProcedureUserGetTVShowEpisodeDownload = "chill.v4.UserService/GetTVShowEpisodeDownload"
	ProcedureUserGetTVShowSeasonDownloads = "chill.v4.UserService/GetTVShowSeasonDownloads"
	ProcedureUserGetUserProfile           = "chill.v4.UserService/GetUserProfile"
	ProcedureUserGetUserSettings          = "chill.v4.UserService/GetUserSettings"
	ProcedureUserSaveUserSettings         = "chill.v4.UserService/SaveUserSettings"
	ProcedureUserAddTransfer              = "chill.v4.UserService/AddTransfer"
	ProcedureUserGetTransfer              = "chill.v4.UserService/GetTransfer"
	ProcedureUserGetIndexers              = "chill.v4.UserService/GetIndexers"
	ProcedureUserGetDownloadFolder        = "chill.v4.UserService/GetDownloadFolder"
	ProcedureUserGetFolder                = "chill.v4.UserService/GetFolder"
)
