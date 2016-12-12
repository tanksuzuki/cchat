<msginput>
    <form class="is-marginless" onsubmit={send}>
        <div class="control">
            <p class="control is-expanded">
              <input class="input {is-disabled: loading}" type="text" name="input" placeholder="Type something" onkeyup={edit}>
            </p>
        </div>
    </form>

    <script>
      edit(e) {
          this.msg = e.target.value;
      }

      send(e) {
          if (this.msg) {
              this.loading = true;
              let self = this;
              superagent.post("/api/message").type('form').send({body: this.msg}).end(function(err, res){
                self.update({loading: false});
                if (err) throw err;
                self.input.value = "";
                self.update({msg: ""});
                obs.trigger("fetch");
              });
          }
      }
    </script>
</msginput>
