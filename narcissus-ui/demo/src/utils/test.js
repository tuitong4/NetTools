
import utils from './utils.js'
import axios from 'axios'
import { get_Hisurl } from "./eastmoney.js";

export class Test {
    constructor(download_data_func) {
        this.args = {}
        this.download_data_func = download_data_func
    }

    get_val() {

        let params = {
            fields1: "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
            fields2: "f51,f53,f58",
            ndays: 1,
            secid: '1.000300',
            iscr: 0
        };
        //callback("hs300_trade", data, false)
        let history_url = get_Hisurl() + "api/qt/stock/trends2/get" + "?" + utils.paramStringify(params)
        //console.log(syncGet(history_url))
        // axios.get(history_url).then((res) => {
        //     let obj = res.data
        //     if (obj.rc == 0 && obj.data) {
        //         obj.data.trends.forEach((v) => {
        //             let items = v.split(",")
        //             let _time = items[0].split(" ")[1]
        //             let idx = utils.timeline[_time]
        //             data[idx]["price"] = parseFloat(parseFloat(items[1]).toFixed(2))
        //             //data[idx]["price_mean"] = parseFloat(parseFloat(items[2]).toFixed(2))
        //         })
        //         callback("hs300_trade", data, false, { "index_pre_day": 1234 })
        //     }
        // }).catch((err) => { console.log(err) })

        // let _res = await axios.get(history_url)
        // let obj = _res.data
        // this.args = obj
        axios.get(history_url).then((res) => {
            let obj = res.data
            this.args = obj
        })

    }

    async download_data() {
        await this.download_data_func((data) => {
            this.args = data
        })
    }
}

async function get_val_3rd(setData) {
    let params = {
        fields1: "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
        fields2: "f51,f53,f58",
        ndays: 1,
        secid: '1.000300',
        iscr: 0
    };

    let history_url = get_Hisurl() + "api/qt/stock/trends2/get" + "?" + utils.paramStringify(params)

    await axios.get(history_url).then((res) => {
        let obj = res.data
        setData(obj)
    })
}

export async function testFunc() {
    let t = new Test();
    t.download_data_func = get_val_3rd
    await t.download_data();
    console.log("test", t.args);
}