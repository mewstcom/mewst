# typed: false
# frozen_string_literal: true

class Latest::PostSerializer < Latest::ApplicationSerializer
  root_key :post, :posts

  attributes :id, :comment, :published_at, :stamps_count, :viewer_has_stamped
  attributes

  one :profile, resource: Latest::ProfileSerializer
end
