# typed: strict
# frozen_string_literal: true

class Dropdowns::PostOptionsDropdownComponent < ApplicationComponent
  sig { params(post: Post).void }
  def initialize(post:)
    @post = post
  end
end
