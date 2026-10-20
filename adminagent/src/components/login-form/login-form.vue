<template>
  <Form ref="loginForm" :model="form" :rules="rules" @keydown.enter.native="handleSubmit">
    <FormItem prop="name">
      <i-input v-model="form.name" placeholder="请输入用户名">
        <span slot="prepend">
          <Icon :size="16" type="ios-person"></Icon>
        </span>
      </i-input>
    </FormItem>
    <FormItem prop="password">
      <i-input type="password" v-model="form.password" placeholder="请输入密码">
        <span slot="prepend">
          <Icon :size="14" type="md-lock"></Icon>
        </span>
      </i-input>
    </FormItem>
    <FormItem>
      <Row>
        <i-col span="15">
          <i-input v-model="validInput" placeholder="请输入验证码"></i-input>
        </i-col>
        <i-col span="7">
          <ValidCode style="transform: scale(0.8);" v-if="modalShow" @input="validCode" :value.sync="validCode"></ValidCode>
        </i-col>
      </Row>
    </FormItem>
    <FormItem>
      <Button @click="handleSubmit" type="primary" long>登录</Button>
    </FormItem>
  </Form>
</template>
<script>
  // import { login } from "@/api/user";
  import ValidCode from "_c/valid-code";
  export default {
    name: "LoginForm",
    components: {
      ValidCode
    },
    props: {
      userNameRules: {
        type: Array,
        default: () => {
          return [{
            required: true,
            message: "账号不能为空",
            trigger: "blur"
          }];
        }
      },
      passwordRules: {
        type: Array,
        default: () => {
          return [{
            required: true,
            message: "密码不能为空",
            trigger: "blur"
          }];
        }
      }
    },
    data() {
      return {
        form: {
          name: "",
          password: ""
        },
        validInput: "",
        validVal: "",
        modalShow: true
      };
    },
    computed: {
      rules() {
        return {
          name: this.userNameRules,
          password: this.passwordRules
        };
      }
    },
    methods: {
      validCode(data) {
        this.validVal = data;
        // comment up when publish
        // this.validInput = data;
      },
      refreshCode() {
        this.modalShow = false;
        setTimeout(() => {
          this.modalShow = true;
        }, 500);
      },
      handleSubmit() {
        if (this.validVal.toUpperCase() === this.validInput.toUpperCase()) {
        this.$refs.loginForm.validate(valid => {
          if (valid) {
            this.$emit("on-success-valid", {
              name: this.form.name,
              password: this.form.password
            });
          } else {
            this.refreshCode();
          }
        });
        } else {
          this.$Message.error("验证码错误");
          this.refreshCode();
        }
      }
    }
  };

</script>
