import utils from './utils.js'
import axios from 'axios'

export function stockSearch(key, set_args) {
    let url = 'http://www.zsxg.cn/api/v2/capital/searchV3'
    let params = {
        text: key
    }
    axios.get(url + "?" + utils.paramStringify(params)).then((res) => {
        let obj = res.data
        set_args(obj)
    }).catch((err) => { console.log(err) })
}