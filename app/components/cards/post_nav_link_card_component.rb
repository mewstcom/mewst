# typed: strict
# frozen_string_literal: true

module Cards
  class PostNavLinkCardComponent < ApplicationComponent
    sig { params(post: Post).void }
    def initialize(post:)
      @post = post
    end

    sig { returns(Post) }
    attr_reader :post
    private :post
  end
end
