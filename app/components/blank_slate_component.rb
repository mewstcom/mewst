# typed: strict
# frozen_string_literal: true

class BlankSlateComponent < ApplicationComponent
  sig { params(message: String, class_name: String).void }
  def initialize(message:, class_name: "")
    @message = message
    @class_name = class_name
  end

  sig { returns(String) }
  attr_reader :message
  private :message

  sig { returns(String) }
  private def class_name
    class_list = %w[c-blank-slate]
    class_list << @class_name if @class_name.present?
    class_list.join(" ")
  end
end
