# typed: strict
# frozen_string_literal: true

class Cards::LinkCardComponent < ApplicationComponent
  sig { params(link: Link).void }
  def initialize(link:)
    @link = link
  end

  sig { returns(Link) }
  attr_reader :link
  private :link
end
