package ws

import (
    "testing"
     "net/http"
)


func  Test_GetMsg_404(t *testing.T) {
    NewWebService("localhost", 7777).Run()
    resp, err := http.Get("http://localhost:7777/topicdoesntexist/sub")

    if err != nil {
       t.Fail()
    }

    if resp.StatusCode != http.StatusNotFound {
        t.Fail()
    }

}

// obviously more tests needed