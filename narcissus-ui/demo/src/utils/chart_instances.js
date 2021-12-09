
import utils from './utils.js'
//import { Chart } from "@antv/g2";
import axios from 'axios'
import { get_url, get_Hisurl } from "./eastmoney.js";
import { FinacialChart } from './chart.js'

function init_hs300_timeline() {
    let data = []
    for (let key in utils.timeline) {
        data.push({
            time: key,
            price: null,
            price_mean: null
            // price: 0,
            // price_mean: 0
        })
    }
    return data
}

function init_north_capital_timeline() {
    let data = []
    for (let key in utils.timeline_type2) {
        data.push({
            time: key,
            capital_sz: null,
            capital_sh: null,
            capital_total: null,
        })
    }
    return data
}

// function set_timeline_data(orignal_data, new_data) {
//     //orignal_data: Array
//     //new_data: json
//     new_data.forEach((key) => {
//         idx = utils.timeline[key]
//         orignal_data[i]["price"] = new_data[key]["price"]
//         orignal_data[i]["price_mean"] = new_data[key]["price_mean"]
//     })
// }

// function syncGet(url) {
//     let xmlhttp = new XMLHttpRequest();
//     xmlhttp.open("GET", url, false);
//     xmlhttp.send(null);
//     return JSON.parse(xmlhttp.responseText)
// }

export function init_hs300_quote() {
    let view_props = {
        "index": {
            getDataFunc: async function (set_chart_args) {
                let _url = get_url();
                let params = {
                    fields1: "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
                    fields2: "f51,f53,f58",
                    ndays: 1,
                    secid: '1.000300',
                    iscr: 0
                };

                let data = init_hs300_timeline()
                let history_url = get_Hisurl() + "api/qt/stock/trends2/get" + "?" + utils.paramStringify(params)

                let _res = await axios.get(history_url)
                let obj = _res.data
                if (obj.rc == 0 && obj.data) {
                    obj.data.trends.forEach((v) => {
                        let items = v.split(",")
                        let _time = items[0].split(" ")[1]
                        let idx = utils.timeline[_time]
                        data[idx]["price"] = parseFloat(parseFloat(items[1]).toFixed(2))
                        //data[idx]["price_mean"] = parseFloat(parseFloat(items[2]).toFixed(2))
                    })
                    set_chart_args("index", data, false, { "index_pre_day": obj.data.preClose })
                } else {
                    set_chart_args("index", data, false, { "index_pre_day": 1 })
                }

                let full_url = _url + "api/qt/stock/trends2/sse" + "?" + utils.paramStringify(params)
                let evtSource = new EventSource(full_url);
                evtSource.onmessage = function (msg) {
                    let obj = JSON.parse(msg.data);
                    if (obj.rc == 0 && obj.data) {
                        if (obj.data.beticks) {
                            return
                        }
                        obj.data.trends.forEach((v) => {
                            let items = v.split(",")
                            let _time = items[0].split(" ")[1]
                            let idx = utils.timeline[_time]
                            data[idx]["price"] = parseFloat(parseFloat(items[1]).toFixed(2))
                            //data[idx]["price_mean"] = parseFloat(parseFloat(items[2]).toFixed(2))
                        })
                        //if (obj.data.beticks) {
                        //    callback("index", data, true, { "index_pre_day": obj.data.preClose })
                        //} else {
                        set_chart_args("index", data, true)
                        //}

                    }
                };
            }
        },
        "north_capital_flow": {
            getDataFunc: function (set_chart_args) {
                let _url = get_url();
                let params = {
                    fields1: "f1,f3",
                    fields2: "f51,f52,f54,f56"
                };

                let data = init_north_capital_timeline()
                //callback("north_capital_flow", data, false)
                let full_url = _url + "api/qt/kamt.rtmin/get" + "?" + utils.paramStringify(params)
                axios.get(full_url).then((res) => {
                    let obj = res.data
                    if (obj.rc == 0 && obj.data) {
                        obj.data.s2n.forEach((v) => {
                            let items = v.split(",")
                            let _time = items[0]
                            let idx = utils.timeline_type2[_time]
                            data[idx]["capital_sh"] = parseFloat(parseFloat(items[1] / 10000).toFixed(2))
                            data[idx]["capital_sz"] = parseFloat(parseFloat(items[2] / 10000).toFixed(2))
                            data[idx]["capital_total"] = parseFloat(parseFloat(items[3] / 10000).toFixed(2))
                        })
                        set_chart_args("north_capital_flow", data, true)
                    }
                }).catch((err) => { console.log(err) })

                setInterval(() => {
                    axios.get(full_url).then((res) => {
                        let obj = res.data
                        if (obj.rc == 0 && obj.data) {
                            obj.data.s2n.forEach((v) => {
                                let items = v.split(",")
                                let _time = items[0]
                                let idx = utils.timeline_type2[_time]
                                data[idx]["capital_sh"] = parseFloat(parseFloat(items[1] / 10000).toFixed(2))
                                data[idx]["capital_sz"] = parseFloat(parseFloat(items[2] / 10000).toFixed(2))
                                data[idx]["capital_total"] = parseFloat(parseFloat(items[3] / 10000).toFixed(2))
                            })
                            set_chart_args("north_capital_flow", data, true)
                        }
                    }).catch((err) => { console.log(err) })
                }, 10000)
            }
        }
    }
    let chart = new FinacialChart({ container: "quote_container", autoFit: true, height: 400, padding: [20, 20, 20, 20] },
        view_props)

    return chart
}
