<list>
  <article each={items} class="media">
      <figure class="media-left">
          <p class="image profileimg">
              <img src="https://www.gravatar.com/avatar/{md5(owner)}?d=mm">
          </p>
      </figure>
      <div class="media-content">
          <div class="content">
              <p>
                  <strong>{owner}</strong> <small><span class="time" datetime="{date}"></span></small>
                  <br> {emoji(body)}
              </p>
          </div>
      </div>
  </article>

  <script>
    let self = this;
    let timeAgo = new timeago();
    let emoji = new EmojiConvertor();

    this.fetch = function() {
      superagent.get("/api/messages").end(function(err, res){
        if (err) throw err;
        self.update({items: res.body});
      });
    }

    this.emoji = function(text) {
      return emoji.replace_colons(text)
    }

    this.fetch();
    setInterval(this.fetch, 30000);

    this.on("updated", function() {
      timeAgo.render(document.querySelectorAll('.time'));
    })

    obs.on("fetch", function() {
      setTimeout(self.fetch, 2000);
    });
  </script>

</list>
