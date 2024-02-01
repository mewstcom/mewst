atom_feed(root_url: profile_url(atname: @profile.atname)) do |feed|
  feed.title("#{@profile.name_with_atname} | Mewst")

  feed.updated(@posts[0].published_at) if @posts.size > 0

  @posts.each do |post|
    feed.entry(post, url: false) do |entry|
      entry.link(href: profile_post_url(atname: @profile.atname, post_id: post.id), rel: "alternate", type: "text/html")

      entry.title(post.content.truncate(50))
      entry.content(post.content, type: "text")

      feed.published(post.published_at)
      feed.updated(post.published_at)

      entry.author do |author|
        author.name(@profile.name_with_atname)
      end
    end
  end
end
