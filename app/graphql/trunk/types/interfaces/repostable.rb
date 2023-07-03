# typed: strict
# frozen_string_literal: true

module Trunk::Types::Interfaces::Repostable
  include Trunk::Types::Interfaces::Base

  orphan_types Trunk::Types::Objects::CommentedPostType, Trunk::Types::Objects::CommentedRepostType
end
