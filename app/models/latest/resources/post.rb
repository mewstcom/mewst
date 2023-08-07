# typed: false
# frozen_string_literal: true

class Latest::Resources::Post < Latest::Resources::Base
  root_key :post, :posts

  attributes :id, :kind, :published_at, :reposts_count

  attribute :postable do |post|
    if post.kind_comment_post?
      Latest::Resources::CommentPost.new(post.comment_post!).to_h
    elsif post.kind_repost?
      Latest::Resources::Repost.new(post.repost!).to_h
    else
      fail
    end
  end

  one :profile, resource: Latest::Resources::Profile
end
