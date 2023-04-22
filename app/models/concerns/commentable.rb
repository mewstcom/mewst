# typed: strict
# frozen_string_literal: true

module Commentable
  extend T::Sig
  extend ActiveSupport::Concern

  MAXIMUM_COMMENT_LENGTH = 500
end
