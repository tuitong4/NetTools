function randomNum() {
    let getParam = function (name) {
        let urlpara = location.search;
        let par = {};
        if (urlpara != "") {
            urlpara = urlpara.substring(1, urlpara.length);
            let para = urlpara.split("&");
            let parname;
            let parvalue;
            for (let i = 0; i < para.length; i++) {
                parname = para[i].substring(0, para[i].indexOf("="));
                parvalue = para[i].substring(para[i].indexOf("=") + 1, para[i].length);
                par[parname] = parvalue;
            }
        }
        if (typeof (par[name]) != "undefined") {
            return par[name];
        } else {
            return null;
        }
    };

    if (getParam('num')) {
        return getParam('num');
    } else {
        return Math.floor(Math.random() * 100 + 1)
    }
}

module.exports = {
    get_url: function () {
        return "//" + randomNum() + ".push2.eastmoney.com/";
    },
    get_Hisurl: function () {
        return "//push2his.eastmoney.com/";
    },

    get_FutureUrl: function () {
        return "//datainterface.eastmoney.com/EM_DataCenter/JS.aspx"
    }
}