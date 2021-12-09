import axios from 'axios'

export function getNorthCapitalTransData(set_args) {
    //com_exchange is string
    let url = "/api/hsgttrans"

    axios.get(url).then((res) => {
        //let data = Array()
        let obj = res.data
        let data = {}
        let market = ""
        for (market in obj) {
            data[market] = []
            obj[market].forEach((el) => {
                let _data = {}
                _data["rank"] = el.RANK;
                _data["stock_name"] = el.SECURITY_NAME;
                _data["price_close"] = el.CLOSE_PRICE;
                _data["price_changed_pct"] = el.CHANGE_RATE;
                _data["volume"] = el.NET_BUY_AMT / 100000000

                data[market].push(_data)
            });
        }

        let _trans_date = obj[market][0].TRADE_DATE.split(" ")[0]
        if (!_trans_date) {
            _trans_date = "N/A"
        }
        set_args({
            trans_date: _trans_date,
            data: data
        })
    }).catch((err) => { console.log(err) })

}