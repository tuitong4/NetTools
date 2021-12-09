import axios from 'axios'
import { Area, Line, DualAxes } from "@antv/g2plot";
import utils from './utils.js'

export function getMainIndicatrixData(type, stock_code, years, set_args) {
    let url = "api/mainindicatrix/" + type

    if (!years) {
        years = "6"
    }

    let params = {
        stock_code: stock_code,
        years: years
    }

    axios.get(url + "?" + utils.paramStringify(params)).then((res) => {
        let obj = res.data
        set_args(obj)
    }).catch((err) => { console.log(err) })

}


export function init_StockBusinessAreaChart(container_id, data, yfield) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }
    let chart = new Area(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            //isStack: true,
            xField: "time",
            yField: yfield,
            seriesField: "name",
            yAxis: {
                grid: null, label: {
                    formatter: (value) => {
                        return parseFloat(value).toFixed(0)
                    }
                }
            },
            legend: {
                position: 'top'
            },
            tooltip: {
                formatter: (item) => {
                    return { name: item.name, value: item[yfield] + '%' };
                }
            },
            color: ["#B2DFDB", '#D4E157', '#FB8C00', '#F4511E', '#BA68C8'],
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 20, 20, 40]
        })
    return chart

}

export function init_StockBusinessCurveChart(container_id, data, yfield) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }
    let chart = new Line(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            //smooth: true,
            xField: "time",
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
                        return parseFloat(value).toFixed(0)
                    }
                }
            },
            legend: {
                position: 'top'
            },
            tooltip: {
                formatter: (item) => {
                    return { name: item.name, value: item[yfield] + '%' };
                }
            },
            color: ["#B2DFDB", '#D4E157', '#FB8C00', '#F4511E', '#BA68C8'],
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 50, 20, 40]
        })
    return chart

}

export function init_StockValuationCurveChart(container_id, data, min_value, mid_value, max_value) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }
    let chart = new Area(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            smooth: true,
            xField: "time",
            yField: "value",
            yAxis: {
                grid: null, label: {
                    formatter: (value) => {
                        return parseFloat(value).toFixed(0)
                    }
                }
            },
            xAxis: {
                tickCount: 4
            },
            legend: {
                position: 'top'
            },
            annotations: [
                {
                    type: 'line',
                    start: ['min', min_value],
                    end: ['max', min_value],
                    style: {
                        stroke: '#44a76f',
                        lineDash: [2, 2],
                    },
                },
                {
                    type: 'line',
                    start: ['min', mid_value],
                    end: ['max', mid_value],
                    style: {
                        stroke: '#FFCA28',
                        lineDash: [2, 2],
                    },
                },
                {
                    type: 'line',
                    start: ['min', max_value],
                    end: ['max', max_value],
                    style: {
                        stroke: '#F4511E',
                        lineDash: [2, 2],
                    },
                },
                {
                    type: 'text',
                    position: ['max', min_value],
                    content: min_value,
                    offsetX: 4,
                    style: {
                        fill: '#44a76f'
                    },
                },
                {
                    type: 'text',
                    position: ['max', mid_value],
                    content: mid_value,
                    offsetX: 4,
                    style: {
                        fill: '#FFCA28'
                    },
                },
                {
                    type: 'text',
                    position: ['max', max_value],
                    content: max_value,
                    offsetX: 4,
                    style: {
                        fill: '#F4511E'
                    },
                },
            ],

            tooltip: {
                formatter: (item) => {
                    return { name: "估值", value: item.value };
                }
            },
            color: "#B2DFDB",
            //color: ["#B2DFDB", '#D4E157', '#FB8C00', '#F4511E', '#BA68C8'],
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 50, 20, 40]
        })
    return chart

}

export function init_StockNortCapitalTrendsCurveChart(container_id, data) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }

    //let capital_data = data.filter(currentValue => currentValue.name === "北向资金")
    //let price_data = data.filter(currentValue => currentValue.name === "股价")

    let chart = new DualAxes(container_id,
        {
            data: [data, data],
            autoFit: true,
            height: height,
            xField: "time",
            yField: ["price", "rate"],
            meta: {
                price: {
                    alias: '股价',
                },
                rate: {
                    alias: '北向资金占比(%)',
                },

            },
            xAxis: {
                tickCount: 4
            },
            yAxis: {
                price: { grid: null },
                rate: { grid: null },
            },
            geometryOptions: [
                {
                    geometry: 'line',
                    color: "#B2DFDB",
                },
                {
                    geometry: 'line',
                    color: '#FB8C00'
                },
            ],
            legend: {
                position: 'top'
            },
            // tooltip: {
            //     formatter: (item) => {
            //         console.log(item)
            //         return { name: item.name, value: item.value };
            //     }
            // },
            // tooltip: {
            //     fields: ['name', 'value'],
            //     formatter: (item) => {
            //         return { name: item.name, value: item.value };
            //     }
            // },
            color: ["#B2DFDB", '#D4E157', '#FB8C00', '#F4511E', '#BA68C8'],
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 50, 20, 40]
        })
    return chart

}

export function init_StockProfitabilityCurveChart(container_id, data) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 150
    }
    let chart = new Line(container_id,
        {
            data: data,
            autoFit: true,
            height: height,
            //smooth: true,
            xField: "time",
            yField: "value",
            label: {
                style: {
                    fill: '#4DB6AC',

                },
                formatter: (item) => {
                    return parseFloat(item.value).toFixed(2) + '%'
                }
            },
            point: {
                size: 5,
                style: {
                    fill: '#4DB6AC',
                    lineWidth: 1,
                    fillOpacity: 1,
                }
            },
            yAxis: {
                grid: null, label: {
                    formatter: (value) => {
                        return parseFloat(value).toFixed(0)
                    }
                }
            },
            xAxis: {
                tickCount: 4
            },
            legend: {
                position: 'top'
            },

            tooltip: {
                formatter: (item) => {
                    return { name: "值", value: parseFloat(item.value).toFixed(2) + "%" };
                }
            },
            color: "#B2DFDB",
            //color: ["#B2DFDB", "#80CBC4", "#4DB6AC", "#26A69A"],
            padding: [30, 50, 20, 40]
        })
    return chart

}