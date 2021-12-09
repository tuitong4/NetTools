
import { Chart } from "@antv/g2";

export class FinacialChart extends Chart {
    //chart_props 是父类的构造函数参数
    //views_props 是关于view的参数信息，包括数据获取渠道，view名等等
    /*
    views_props = {
        "view_one":{
            getDataFunc: function(callback_func),
            }
        }
    */
    constructor(chart_props, views_props) {
        super(chart_props)
        this.view_props = views_props
        this._views = {}
        this.other_args = {} //存储一些自定义的变量
    }

    initViews() {
        for (let v_name in this.view_props) {
            this._views[v_name] = this.createView()
        }
    }

    _setViewData(view_name, data, is_update) {
        //update=false初始化数据
        //update=true则更新数据
        if (is_update) {
            this._views[view_name].changeData(data)
            return
        }
        this._views[view_name].data(data)

    }

    async _getViewData(view_name) {
        await this.view_props[view_name].getDataFunc((view, data, is_update, other_args) => {
            this._setViewData(view, data, is_update)
            if (other_args) {
                for (let key in other_args) {
                    this.other_args[key] = other_args[key]
                }
            }
        })
    }

    async fetchData() {
        for (let v_name in this._views) {
            await this._getViewData(v_name)
        }
    }

    getOtherArgs(key) {
        return this.other_args[key]
    }
}


// view getDataFunc demo
//---------------------------------
/*
function view_get_data(func) {
    //获取数据
    let _data = [1, 2, 3]

    //处理数据
    let data = _data.forEach((v) => { v = v + 10 })

    //设置设局的操作法师，更新或者初始化
    let update = true
    //处理回调
    func(view_name, data, update)

    //其他处理
}
*/
