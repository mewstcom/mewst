# typed: strict
# frozen_string_literal: true

class Resources::Post < Resources::Base
  root_key :post, :posts

  attributes :id, :published_at, :reposts_count

  one :postable, resource: ->(record) {
    case record
    when CommentedPost
      Resources::CommentedPost
    else
      fail
    end
  }

  attribute :postable_type do |post|
    post.postable_type.underscore
  end

  one :profile, resource: Resources::Profile
end
