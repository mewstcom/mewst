# typed: false
# frozen_string_literal: true

class Latest::CommentPostSerializer < Latest::ApplicationSerializer
  root_key :comment_post, :comment_posts

  attributes :comment
end
