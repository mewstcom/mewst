# typed: false
# frozen_string_literal: true

class Resources::Latest::Post < Resources::Latest::Base
  root_key :post, :posts

  attributes :id, :kind, :published_at, :reposts_count

  attribute :postable do |post|
    if post.kind_comment_post?
      Resources::Latest::CommentPost.new(post.comment_post!).to_h
    elsif post.kind_repost?
      Resources::Latest::Repost.new(post.repost!).to_h
    else
      fail
    end
  end

  one :profile, resource: Resources::Latest::Profile
end
