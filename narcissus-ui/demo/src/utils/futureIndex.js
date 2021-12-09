import axios from 'axios'

export function getFutureData(com_exchange, set_args) {
    //com_exchange is string
    let url = "/api/futureposition/" + com_exchange
    axios.get(url).then((res) => {
        //let data = Array()
        let obj = res.data

        set_args(obj)
    }).catch((err) => { console.log(err) })

}

