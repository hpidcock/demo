<template>
  <v-row>
    <v-col cols="12" sm="6" offset-sm="3">
      <v-card>
        <v-card-title class="headline">
          k8s demo
        </v-card-title>
        <v-form
          ref="form"
        >
          <v-container>
            <v-row>
              <v-col
                cols="12"
                sm="8"
              >
                <v-file-input
                  v-model="file"
                  label="Upload Image"
                  filled
                  prepend-icon="mdi-camera"
                ></v-file-input>
              </v-col>
              <v-col
                cols="12"
                sm="4"
              >
                <v-btn
                  color="success"
                  class="mr-4"
                  @click="upload"
                >
                  Upload
                </v-btn>
              </v-col>
            </v-row>
          </v-container>
        </v-form>
        <v-btn
          :disabled="disabled"
          color="success"
          class="mr-4"
          @click="convert"
        >
          Convert
        </v-btn>
        <v-col cols="12" sm="12" offset-sm="0">
          <v-container fluid>
            <v-row>
              <v-col
                v-for="n in images"
                :key="n"
                class="d-flex child-flex"
                cols="4"
              >
                <v-card flat tile class="d-flex">
                  <v-img
                    :src="`/api/images/${n}`"
                    aspect-ratio="1"
                    v-bind:class="{ targetimage: target==n, styleimage: style==n, grey: target!=n&&style!=n, 'deep-purple': target==n, 'deep-orange': style==n }"
                    @click="selectConvert(n)"
                  >
                    <template v-slot:placeholder>
                      <v-row
                        class="fill-height ma-0"
                        align="center"
                        justify="center"
                      >
                        <v-progress-circular indeterminate color="grey lighten-5"></v-progress-circular>
                      </v-row>
                    </template>
                  </v-img>
                </v-card>
              </v-col>
            </v-row>
          </v-container>
        </v-col>
      </v-card>
    </v-col>
  </v-row>
</template>

<style scoped>
  .styleimage {
    border-style: solid;
  }
  .targetimage {
    border-style: solid;
  }
</style>

<script>

export default {
  components: {},

  mounted() {
    window.setInterval(this.fetchImages, 1000);
  },

  data() {
    return {
      images: [],
      file: null,
      target: null,
      style: null,
    }
  },

  computed: {
    disabled() {
      return this.target == null || this.style == null
    },
  },

  methods: {
    async fetchImages() {
      this.images = await this.$axios.$get('/api/images')
    },
    async upload() {
      let formData = new FormData()
      formData.append('file', this.file, this.file.name)
      await this.$axios.post('/api/images/upload', formData, {
        headers: {
          'Content-Type': 'multipart/form-data'
        }
      })
    },
    selectConvert(id) {
      if(this.target != null && this.style != null){
        this.target = null
        this.style = null
      }
      if(this.target == id && this.style == null) {
        this.target = null
      }
      if(this.target == null) {
        this.target = id
      } else if(this.style == null) {
        this.style = id
      }
    },
    async convert() {
      await this.$axios.post('/api/convert', {
        'image-id': this.target,
        'style-id': this.style,
      })
      this.target = null
      this.style = null
    }
  }
}
</script>
