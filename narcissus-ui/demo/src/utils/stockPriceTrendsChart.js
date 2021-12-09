
import { Mix } from "@antv/g2plot";
import utils from './utils.js'
//import { Chart } from "@antv/g2";
import axios from 'axios'
import { get_Hisurl } from "./eastmoney.js"

// export function init_mainMetricChart() {

//     let chart = new Column("main_metric_container")


//     return chart
// }

export function init_StockPriceTrendsChart(stock_data, index_data, stock_name, index_name) {

    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 100
    }
    let _padding = [30, 50, 20, 40]
    let chart = new Mix("stock_trends_container",
        {
            appendPadding: 8,
            tooltip: { shared: true },
            syncViewPadding: true,
            autoFit: true,
            height: height,
            plots: [
                {
                    type: 'area',
                    options: {
                        data: stock_data,
                        xField: 'date',
                        yField: 'price',
                        yAxis: { grid: null, label: null, tickCount: 3, tickMethod: 'quantile' },
                        xAxis: { tickCount: 0 },
                        color: '#FF5252',
                        areaStyle: {
                            fill: "l(270) 0:#ffffff 0.5:#FF5252 1:#FF5252",
                            fillOpacity: 0.1
                        },
                        meta: {
                            price: {
                                alias: stock_name,
                            },
                        },
                        padding: _padding
                    },
                },
                {
                    type: 'area',
                    options: {
                        data: index_data,
                        xField: 'date',
                        yField: 'price',
                        yAxis: { grid: null, label: null, tickCount: 3, tickMethod: 'quantile', minLimit: 0 },
                        xAxis: { tickCount: 5 },
                        color: '#66BB6A',
                        areaStyle: {
                            fill: "l(270) 0:#ffffff 0.5:#66BB6A 1:#66BB6A",
                            fillOpacity: 0.1
                        },
                        meta: {
                            price: {
                                alias: index_name,
                            },
                        },
                        padding: _padding
                    },
                }]
        })
    return chart
}


export async function getStockPriceTrends(secid, set_args) {
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

    let _stcok_data = []
    let _index_data = []
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
            _stcok_data.push(
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
            _index_data.push(
                {
                    date: date,
                    price: index_price,
                    name: index_name
                }
            )
        })
        set_args(_stcok_data, _index_data, stock_obj.data.code, stock_obj.data.name, index_name)
    }

}