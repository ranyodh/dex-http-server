package main

//
//type auth struct {
//	http.ResponseWriter
//}
//
//func (rsp *auth) WriteHeader(code int) {
//	rsp.ResponseWriter.WriteHeader(code)
//}
//
//// Unwrap returns the original http.ResponseWriter. This is necessary
//// to expose Flush() and Push() on the underlying response writer.
//func (rsp *auth) Unwrap() http.ResponseWriter {
//	return rsp.ResponseWriter
//}
//
//func newAuthHandler(w http.ResponseWriter) *auth {
//	return &auth{w, http.StatusOK}
//}
