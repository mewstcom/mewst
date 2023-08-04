# typed: false
# frozen_string_literal: true

class Resources::Latest::Post < Resources::Latest::Base
  root_key :post, :posts

  attributes :id, :published_at, :reposts_count, :stamps_count, :viewer_has_stamped

  one :postable, resource: ->(record) {
    case record
    when CommentedPost
      Resources::Latest::CommentedPost
    else
      fail
    end
  }

  attribute :postable_type do |post|
    post.postable_type.underscore
  end

  one :profile, resource: Resources::Latest::Profile
end
