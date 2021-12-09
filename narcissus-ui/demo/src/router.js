import Vue from 'vue';
import VueRouter from 'vue-router'

Vue.use(VueRouter);

export default new VueRouter({
    mode: 'hash',
    base: process.env.BASE_URL,
    routes: [

        {
            path: '/',
            component: () => import('@/views/Index'),
            redirect: '/mainquote'
        },

        {
            path: '/mainquote',
            component: () => import('@/views/MainQuote')
        },

        {
            path: '/stock',
            component: () => import('@/views/Stock')
        },
        // {
        //     path: '/stockmainindex',
        //     component: () => import('@/views/StockMainIndicatrix')
        // },
        // {
        //     path: '/test',
        //     component: () => import('@/views/Test')
        // }
    ],
})