# typed: false
# frozen_string_literal: true

class Resources::Latest::CommentedPost < Resources::Base
  root_key :commented_post, :commented_posts

  attributes :comment
end
