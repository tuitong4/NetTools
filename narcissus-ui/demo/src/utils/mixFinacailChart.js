
import { Chart } from "@antv/g2";
import utils from './utils.js'
//import { Chart } from "@antv/g2";
import axios from 'axios'
import { get_url, get_Hisurl } from "./eastmoney.js";


function init_index_timeline() {
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

function checkTradingTime() {
    let t = new Date()
    let duration = t.getHours() * 60 + t.getMinutes()
    if ((duration >= 570 && duration <= 690) || (duration >= 780 && duration <= 900)) {
        return true
    }
    return false
}

export class MixFinacialChart extends Chart {

    constructor(chart_props, index_props) {
        super(chart_props)
        this.index_view = undefined
        this.capital_view = undefined
        this.index_props = index_props
        this.index_secid = index_props.secid
        this.index_pre_close = 1
    }

    initViews() {
        this.index_view = this.createView()
        this.capital_view = this.createView()
    }

    _setViewData(view, data, is_update) {
        //update=false初始化数据
        //update=true则更新数据
        if (is_update) {
            view.changeData(data)
            return
        }
        view.data(data)
    }

    async fetchIndexData(set_args) {
        // set_args, 回调函数
        let params = {
            fields1: "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
            fields2: "f51,f53,f58",
            ndays: 1,
            secid: this.index_secid,
            iscr: 0
        };

        let data = init_index_timeline()
        let history_url = get_Hisurl() + "api/qt/stock/trends2/get" + "?" + utils.paramStringify(params)

        let latest_idx = 0 //记录最新值得index
        let _res = await axios.get(history_url)
        let obj = _res.data

        if (obj.rc == 0 && obj.data) {
            obj.data.trends.forEach((v) => {
                let items = v.split(",")
                let _time = items[0].split(" ")[1]
                let idx = utils.timeline[_time]
                data[idx]["price"] = parseFloat(parseFloat(items[1]).toFixed(2))
                //data[idx]["price_mean"] = parseFloat(parseFloat(items[2]).toFixed(2))
                latest_idx = idx
            })
            this.index_pre_close = obj.data.preClose
            set_args(data[latest_idx]["price"], obj.data.preClose)
        }

        this._setViewData(this.index_view, data, false)

        if (!checkTradingTime()) {
            return
        }

        let full_url = get_url() + "api/qt/stock/trends2/sse" + "?" + utils.paramStringify(params)
        let evtSource = new EventSource(full_url);
        evtSource.onmessage = (msg) => {
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
                    latest_idx = idx
                })
                set_args(data[latest_idx]["price"], undefined)
                this._setViewData(this.index_view, data, true)
            }
        };
    }


    async fetchCapitalData(set_args) {
        // set_args, 回调函数
        let params = {
            fields1: "f1,f3",
            fields2: "f51,f52,f54,f56"
        };

        let latest_idx = 0 //记录最新的值得index
        let data = init_north_capital_timeline()
        let full_url = get_url() + "api/qt/kamt.rtmin/get" + "?" + utils.paramStringify(params)

        axios.get(full_url).then((res) => {
            let obj = res.data
            if (obj.rc == 0 && obj.data) {
                for (let i = 0; i < obj.data.s2n.length; i++) {
                    let items = obj.data.s2n[i].split(",")
                    //跳过空置的数据
                    if (items[1] === "-") {
                        break
                    }
                    let _time = items[0]
                    let idx = utils.timeline_type2[_time]
                    data[idx]["capital_sh"] = parseFloat(parseFloat(items[1] / 10000).toFixed(2))
                    data[idx]["capital_sz"] = parseFloat(parseFloat(items[2] / 10000).toFixed(2))
                    data[idx]["capital_total"] = parseFloat(parseFloat(items[3] / 10000).toFixed(2))
                    latest_idx = idx
                }
                this._setViewData(this.capital_view, data, true)
                set_args(data[latest_idx]["capital_total"], data[latest_idx]["capital_sh"], data[latest_idx]["capital_sz"])
            }
        }).catch((err) => { console.log(err) })

        // if (!checkTradingTime()) {
        //     return
        // }
        setInterval(() => {
            axios.get(full_url).then((res) => {
                let obj = res.data
                if (obj.rc == 0 && obj.data) {

                    for (let i = 0; i < obj.data.s2n.length; i++) {
                        let items = obj.data.s2n[i].split(",")
                        //跳过空置的数据
                        if (items[1] === "-") {
                            break
                        }
                        let _time = items[0]
                        let idx = utils.timeline_type2[_time]
                        data[idx]["capital_sh"] = parseFloat(parseFloat(items[1] / 10000).toFixed(2))
                        data[idx]["capital_sz"] = parseFloat(parseFloat(items[2] / 10000).toFixed(2))
                        data[idx]["capital_total"] = parseFloat(parseFloat(items[3] / 10000).toFixed(2))
                        latest_idx = idx
                    }
                    this._setViewData(this.capital_view, data, true)
                    set_args(data[latest_idx]["capital_total"], data[latest_idx]["capital_sh"], data[latest_idx]["capital_sz"])
                }
            }).catch((err) => { console.log(err) })
        }, 10000)
    }
}


export function init_mixFinacailChart(index_params) {
    let height = 400
    if (document.documentElement.clientWidth < 960) {
        height = 220
    }
    let chart = new MixFinacialChart(
        {
            container: "quote_container",
            autoFit: true,
            height: height,
            padding: [20, 20, 30, 20]
        },
        index_params)

    return chart
}