import axios from 'axios'
import { Column, Line } from "@antv/g2plot";
import utils from './utils.js'

export function getStatementsAnalysisData(type, stock_code, periods, set_args) {
    let url = "api/fsanalysis/" + type

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


export function init_StockStatementsAnalysisCloumnChart(container_id, data, yfield) {
    if (yfield === undefined) {
        yfield = 'value'
    }
    let height = 300
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
            yField: yfield,
            seriesField: "name",
            yAxis: {
                grid: null, label: {
                    formatter: (value) => {
                        if (value > 100000000 || value < -100000000) {
                            return parseFloat(value / 100000000).toFixed(0)
                        }
                        if (value > 10000 || value < -10000) {
                            return parseFloat(value / 10000).toFixed(0)
                        }
                        return value
                    }
                }
            },
            legend: {
                position: 'top'
            },
            tooltip: {
                formatter: (data) => {
                    if (data.value > 100000000 || data.value < -100000000) {
                        return { name: data.name, value: parseFloat(data.value / 100000000).toFixed(2) + "亿" }
                    }
                    if (data.value > 10000 || data.value < -10000) {
                        return { name: data.name, value: parseFloat(data.value / 10000).toFixed(2) + "万" }
                    }
                    if (!data.value) {
                        return { name: data.name, value: "--" }
                    } else {
                        return { name: data.name, value: parseFloat(data.value).toFixed(2) }
                    }
                }
            },
            color: ["#B2DFDB", '#039BE5', '#1976D2', '#7CB342', '#FB8C00', '#F4511E', '#AD1457', '#6A1B9A', '#8D6E63', '#546E7A'],
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 20, 20, 40]
        })
    return chart
}


export function init_StockStatementsAnalysisCurveChart(container_id, data, yfield) {
    if (yfield === undefined) {
        yfield = 'yoy'
    }

    let height = 300
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }

    let chart = new Line(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            //smooth: true,
            xField: "year",
            yField: yfield,
            seriesField: "name",
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
                        return { name: data.name, value: "--" }
                    } else {
                        return { name: data.name, value: parseFloat(data.yoy).toFixed(2) + '%' };
                    }
                }
            },
            color: ["#B2DFDB", '#039BE5', '#1976D2', '#7CB342', '#FB8C00', '#F4511E', '#AD1457', '#6A1B9A', '#8D6E63', '#546E7A'],
            padding: [30, 20, 20, 40]
        })
    return chart
}

