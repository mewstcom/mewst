# typed: strict
# frozen_string_literal: true

class Forms::PostFormComponent < ApplicationComponent
  sig { params(post_creator: Post::Creator).void }
  def initialize(post_creator:)
    @post_creator = post_creator
  end
end
