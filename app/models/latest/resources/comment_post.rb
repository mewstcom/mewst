# typed: false
# frozen_string_literal: true

class Latest::Resources::CommentPost < Latest::Resources::Base
  root_key :comment_post, :comment_posts

  attributes :comment
end
