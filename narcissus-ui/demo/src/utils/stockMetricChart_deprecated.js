
import { Column, Line } from "@antv/g2plot";
import utils from './utils.js'
//import { Chart } from "@antv/g2";
import axios from 'axios'
import { get_Hisurl } from "./eastmoney.js"

export function init_mainMetricChart() {

    let chart = new Column("main_metric_container")


    return chart
}

export function init_StockTrendsChart(data) {

    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 100
    }

    let chart = new Line("stock_trends_container",
        {
            data: data,
            autoFit: true,
            height: height,
            xField: "date",
            yField: "price",
            seriesField: "name",
            yAxis: { grid: null, tickMethod: 'quantile' },
            legend: {
                //layout: 'vertical',
                position: 'top'
            },
            xAxis: { tickCount: 0 },
            color: ['#FF5252', '#66BB6A', '#FDD835', '#4DD0E1'],
            padding: [30, 20, 20, 30]
        })
    return chart
}




export async function getStockTrends(secid, set_args) {
    let stock_params = {
        fields1: "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
        fields2: "f51,f53",
        end: '20500101',
        secid: secid,
        beg: 0,
        rtntype: 6,
        klt: 101,
        fqt: 1
    };

    let index_params = {
        fields1: "f1,f2,f3,f4,f5,f6,f7,f8,f9,f10,f11,f12,f13",
        fields2: "f51,f53",
        end: '20500101',
        secid: '1.000300',
        beg: 0,
        rtntype: 6,
        klt: 101,
        fqt: 1
    };

    let stock_url = get_Hisurl() + "api/qt/stock/kline/get" + "?" + utils.paramStringify(stock_params)
    let index_url = get_Hisurl() + "api/qt/stock/kline/get" + "?" + utils.paramStringify(index_params)

    let stock_res = await axios.get(stock_url)
    let stock_obj = stock_res.data

    let index_res = await axios.get(index_url)
    let index_obj = index_res.data

    let data = []
    let index_data = {}
    let index_name = ""
    if (index_obj.rc == 0 && index_obj.data) {
        index_name = index_obj.data.name
        index_obj.data.klines.forEach(v => {
            let items = v.split(",")
            index_data[items[0]] = parseFloat(items[1])
        })
    }

    if (stock_obj.rc == 0 && stock_obj.data) {
        stock_obj.data.klines.forEach((v) => {
            let items = v.split(",")
            let date = items[0]
            data.push(
                {
                    date: date,
                    price: parseFloat(items[1]),
                    name: stock_obj.data.name
                }
            )

            let index_price = index_data[date]
            if (index_price === undefined) {
                index_price = null
            }
            data.push(
                {
                    date: date,
                    price: index_price,
                    name: index_name
                }
            )
        })
        set_args(data, stock_obj.data.code, stock_obj.data.name)
    }

}