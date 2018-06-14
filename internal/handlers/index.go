package handlers

import "net/http"

func Index() http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/" {
			http.NotFound(w, r)
			return
		}

		w.WriteHeader(http.StatusOK)
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		w.Write([]byte(`<!DOCTYPE html>
		<html>
		  <head>
			<meta charset="UTF-8">
			<meta http-equiv="X-UA-Compatible" content="IE=edge,chrome=1">
			<meta name="viewport" content="width=device-width,initial-scale=1">
			<title>Nagios Exporter</title>
		  </head>
		  <body>
			<h1>Nagios Exporter</h1>
			<p>
				<ul>
					<li><a href="/metrics">/metrics</a></li>
					<li><a href="/collect?instance=ftmon50799-law1b-eu-p.osb.ft.com">/collect?instance=10.170.37.161</a></li>
					<li><a href="/__gtg">/__gtg</a></li>
				</ul>
			</p>
		  </body>
		</html>`))
	})
}
