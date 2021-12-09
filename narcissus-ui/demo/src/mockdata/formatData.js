/*
* 格式化字符串
*/
function formatStr(value, emptyValue) {
    if (!emptyValue && emptyValue != '') {
        emptyValue = emptyValue || "--";
    }

    if (value == "" || value == null) {
        return emptyValue;
    }

    return value;
}

/*
* 保留小数
*/
function toFixed(value, point, emptyValue) {
    if (!point && point != 0) {
        point = getFormatPoint(value);
    }

    if (!emptyValue && emptyValue != '') {
        emptyValue = emptyValue || "--";
    }
    if (value == "" || value == null) {
        return emptyValue;
    }
    value = parseFloat(value);
    if (isNaN(value)) {
        return emptyValue;
    }

    return value.toFixed(point);
}

/*
* 格式化日期
*/
function formatDate(value, fmt, emptyValue) {
    fmt = fmt || 'yyyy-MM-dd';
    if (!emptyValue && emptyValue != '') {
        emptyValue = emptyValue || "--";
    }
    try {
        var thisDate = new Date(value.replace(/-/g, '/'));
        var weekDay = ["日", "一", "二", "三", "四", "五", "六"];
        var o = {
            "M+": thisDate.getMonth() + 1, //月份
            "d+": thisDate.getDate(), //日
            "h+": thisDate.getHours(), //小时
            "m+": thisDate.getMinutes(), //分
            "s+": thisDate.getSeconds(), //秒
            "q+": Math.floor((thisDate.getMonth() + 3) / 3), //季度
            "S": thisDate.getMilliseconds(), //毫秒
            "W": '周' + weekDay[thisDate.getDay()] //周
        };
        if (/(y+)/.test(fmt)) fmt = fmt.replace(RegExp.$1, (thisDate.getFullYear() + "").substr(4 - RegExp.$1.length));
        for (var k in o)
            if (new RegExp("(" + k + ")").test(fmt)) fmt = fmt.replace(RegExp.$1, (RegExp.$1.length == 1) ? (o[k]) : (("00" + o[k]).substr(("" + o[k]).length)));
        return fmt;
    } catch (e) {
        return emptyValue;
    }
}

/*
* 格式化金额
*/
function formatMoney(value, emptyValue) {
    if (!emptyValue && emptyValue != '') {
        emptyValue = emptyValue || "--";
    }
    if (value == "" || value == null) {
        return emptyValue;
    }
    value = parseFloat(value);
    if (isNaN(value)) {
        return emptyValue;
    }

    var point = 2;
    if (Math.abs(value) >= 1e12) {
        value = value / 1e12;
        point = getFormatPoint(value);
        value = value.toFixed(point) + "万亿";
    } else if (Math.abs(value) >= 1e8) {
        value = value / 1e8;
        point = getFormatPoint(value);
        value = value.toFixed(point) + "亿";
    } else if (Math.abs(value) > 1e4) {
        value = value / 1e4;
        point = getFormatPoint(value);
        value = value.toFixed(point) + "万";
    } else {
        point = getFormatPoint(value);
        value = value.toFixed(point);
    }
    return value;
}

/*
* 获取小数位
*/
function getFormatPoint(value) {
    value = Math.abs(value);
    var point = 2;
    if (value >= 1000)
        point = 0;
    else if (value >= 100)
        point = 1;
    else if (value >= 10)
        point = 2;
    else
        point = 3;

    return point;
}

/*
* 格式化百分比
*/
function formatPercent(value, unit, emptyValue) {
    if (!unit && unit != '') {
        unit = '%';
    }
    if (!emptyValue && emptyValue != '') {
        emptyValue = emptyValue || "--";
    }
    if (value == "" || value == null) {
        return emptyValue;
    }
    value = parseFloat(value);
    if (isNaN(value)) {
        return emptyValue;
    }

    if (Math.abs(value) >= 10000 * 100) {
        return (value / 1e6).toFixed(2) + "万倍";
    } else if (Math.abs(value) >= 100 * 100) {
        return (value / 100).toFixed(2) + "倍";
    } else
        return value.toFixed(2) + unit;
}

function formatNumber(value, point) {
    var temp = Number(value);
    if (!temp)
        return "--";

    if (Math.abs(temp) >= 1e12) {
        var resupt = temp / 1e12;
        point = getFormatPoint(resupt);
        return resupt.toFixed(point) + "万亿";
    }
    else if (Math.abs(temp) >= 1e8) {
        var resupt = temp / 1e8;
        point = getFormatPoint(resupt);
        return resupt.toFixed(point) + "亿";
    }
    else if (Math.abs(temp) >= 1e4) {
        var resupt = temp / 1e4;
        point = getFormatPoint(resupt);
        return resupt.toFixed(point) + "万";
    }
    else {
        if (!point && point != 0) {
            point = getFormatPoint(value);
        }
    }
    return temp.toFixed(point);
}

function formatNumToLocaleStr(value, point) {
    var temp = Number(value);
    if (!temp)
        return "--";

    if (!point && point != 0) {
        point = 2;
    }

    temp = temp.toFixed(point);  //数字转换为字符 
    var str = Number(temp).toLocaleString();
    if (temp.indexOf('.') > 0) {
        str = str.split('.')[0];
        str += temp.substr(temp.indexOf('.'));
    }
    return str;
}

function formatRate(value) {
    var temp = Number(value);
    if (!temp)
        return "--";

    return "<span>" + temp.toFixed(2) + "%</span>";
}

var autocount = 0;
function hideEmptyRow(tableid) {
    var table = document.getElementById(tableid);
    if (!table || table.rows.length == 0)
        return;

    for (var i = 0; i < table.rows.length; i++) {
        if (i <= 0)
            continue;

        var row = table.rows[i];

        if (row.children.length <= 1)
            continue;

        //删除空行
        var length = row.children.length;
        length = length > 9 ? 9 : length;
        var show = false;

        for (var j = 1; j < length; j++) {
            var text = row.children[j].innerText.replace(/\s/g, '');
            if (text != "--") {
                show = true;
                break;
            }
        }

        if (!show) {
            row.hidden = !show;
            row.style.display = "none";
        }

        if (!row.children[0].children || row.children[0].children.length != 2)
            continue;

        //根据父级是否展示，设置展示方式
        var index = 1;
        var isHaveBro = false;
        while (i - index >= 0
            && table.rows[i - index].children.length > 0
            && table.rows[i - index].children[0].children.length > 0) {
            if (table.rows[i - index].children[0].children.length == 2) {
                if (table.rows[i - index].hidden) {
                    //兄弟节点隐藏，继续向上查找
                    index++;
                }
                else {
                    //兄弟节点显示，继续向上查找，并记录
                    index++;
                    isHaveBro = true;
                }
            }
            else {
                if (table.rows[i - index].hidden) {
                    //父级节点隐藏
                    if (index == 1) {
                        //如果只有根节点，不显示'其中'
                        row.children[0].children[0].style.display = "";
                        row.children[0].children[1].style.display = "none";
                    }
                }
                else {
                    //父级节点显示，无兄弟节点显示'其中:'
                    if (!isHaveBro && index != 1) {
                        row.children[0].children[0].style.display = "none";
                        row.children[0].children[1].style.display = "";
                    }
                }

                break;
            }
        }
        //if (i != 0 && table.rows[i - 1].hidden) {
        //    row.children[0].children[0].style.display = "none";
        //    row.children[0].children[1].style.display = "";
        //}
    }

    //定位
    autocount++;
    if (autocount <= 3)
        AutoScroll();
}