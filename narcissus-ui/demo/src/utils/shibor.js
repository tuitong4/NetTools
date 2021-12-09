import axios from 'axios'
import { Line } from "@antv/g2plot";

export function getShiBorData(set_args) {
    let url = "/api/shibor"
    axios.get(url).then((res) => {
        let obj = res.data
        //数据顺序调整，按时间增序
        obj.reverse()
        set_args(obj)
    }).catch((err) => { console.log(err) })

}


export function init_ShiBorChart(container_suffix, data) {
    let height = 200
    if (document.documentElement.clientWidth < 960) {
        height = 100
    }
    let chart = new Line("shibor_container_" + container_suffix,
        {
            data: data,
            autoFit: true,
            height: height,
            xField: "date",
            yField: "value",
            seriesField: "variety",
            yAxis: { grid: null, minLimit: 1, tickCount: 3, tickMethod: 'quantile', label: { formatter: (value) => { return parseFloat(value).toFixed(1) } } },
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