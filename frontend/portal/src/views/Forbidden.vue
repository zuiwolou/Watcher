<template>
  <div class="wrap">
    <h2>403</h2>
    <p>无权限访问该页面</p>
    <el-button type="primary" @click="$router.push('/')">返回工作台</el-button>
  </div>
</template>
<style scoped>
.wrap{padding:48px;text-align:center;color:#334e68}
h2{font-size:48px;margin:0 0 8px;color:#0b3d4a}
</style>
