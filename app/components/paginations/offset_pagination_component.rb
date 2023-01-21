# typed: strict
# frozen_string_literal: true

class Paginations::OffsetPaginationComponent < ApplicationComponent
  include Pagy::Frontend

  sig { params(pagy: Pagy).void }
  def initialize(pagy:)
    @pagy = T.let(pagy, Pagy)
  end
end
