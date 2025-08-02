# typed: strict
# frozen_string_literal: true

module Links
  class PostNavLinkComponent < ApplicationComponent
    class Direction < T::Enum
      enums do
        Prev = new
        Next = new
      end
    end

    sig { params(post: PostRecord, direction: Direction).void }
    def initialize(post:, direction:)
      @post = post
      @direction = direction
    end

    sig { returns(PostRecord) }
    attr_reader :post
    private :post

    sig { returns(Direction) }
    attr_reader :direction
    private :direction

    sig { returns(T::Boolean) }
    def prev?
      direction == Direction::Prev
    end

    sig { returns(T::Boolean) }
    def next?
      direction == Direction::Next
    end

    sig { returns(String) }
    def direction_text
      prev? ? t("nouns.prev_post") : t("nouns.next_post")
    end

    sig { returns(String) }
    def icon_name
      prev? ? "caret-left-regular" : "caret-right-regular"
    end
  end
end
