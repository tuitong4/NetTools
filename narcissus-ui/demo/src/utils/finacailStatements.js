import axios from 'axios'
import { Column, Line } from "@antv/g2plot";
import utils from './utils.js'

export function getStatementData(type, stock_code, periods, set_args) {
    let url = "api/finacialstatements/" + type

    if (!periods) {
        periods = "0331,0630,0930,1231"
    }

    let params = {
        stock_code: stock_code,
        periods: periods
    }

    axios.get(url + "?" + utils.paramStringify(params)).then((res) => {
        let obj = res.data
        set_args(obj)
    }).catch((err) => { console.log(err) })

}

export function init_FinacialStatementsChart(container_id, data) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }
    let chart = new Column(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            isGroup: true,
            xField: "year",
            yField: "value",
            seriesField: "season",
            yAxis: {
                grid: null, label: {
                    formatter: (value) => {
                        return parseFloat(value / 100000000).toFixed(0)
                    }
                }
            },
            legend: {
                position: 'top'
            },
            tooltip: {
                formatter: (data) => {
                    if (data.value > 100000000 || data.value < -100000000) {
                        return { name: data.season, value: parseFloat(data.value / 100000000).toFixed(2) + "亿" }
                    }
                    if (data.value > 10000 || data.value < -10000) {
                        return { name: data.season, value: parseFloat(data.value / 10000).toFixed(2) + "万" }
                    }
                    if (!data.value) {
                        return { name: data.season, value: "--" }
                    }
                }
            },
            color: ["#B2DFDB", '#D4E157', '#FB8C00', '#F4511E'],
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 20, 20, 40]
        })
    return chart
}


export function init_FinacialStatementsCurveChart(container_id, data) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }

    let chart = new Line(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            smooth: true,
            xField: "year",
            yField: "yoy",
            seriesField: "season",
            point: {
                size: 5,
                style: {
                    lineWidth: 1,
                    fillOpacity: 1,
                }
            },

            yAxis: {
                grid: null, label: {
                    formatter: (value) => {
                        return value
                    }
                }
            },
            legend: {
                position: 'top'
            },
            tooltip: {
                formatter: (data) => {
                    if (data.yoy === null) {
                        return { name: data.season, value: "--" }
                    } else {
                        return { name: data.season, value: data.yoy + '%' };
                    }
                }
            },
            color: ["#B2DFDB", '#D4E157', '#FB8C00', '#F4511E'],
            padding: [30, 20, 20, 40]
        })
    return chart
}

