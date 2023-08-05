# typed: false
# frozen_string_literal: true

class Resources::Latest::CommentPost < Resources::Latest::Base
  root_key :comment_post, :comment_posts

  attributes :comment
end
