# typed: strict
# frozen_string_literal: true

module Trunk::Types::Interfaces::Postable
  include Trunk::Types::Interfaces::Base

  orphan_types Trunk::Types::Objects::CommentedPostType
end
